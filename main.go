package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/thedevsaddam/renderer"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var rnd *renderer.Render
var db *mongo.Database

const (
	hostName = "mongodb://127.0.0.1:27017"
	dbName   = "demo_todo"
	collName = "todo"
	port     = ":9000"
)

type (
	todoModel struct {
		ID        primitive.ObjectID `bson:"_id,omitempty"`
		Title     string             `bson:"title"`
		Completed    bool             `bson:"completed"`
		CreatedAt time.Time          `bson:"created_at"`
		UpdatedAt time.Time          `bson:"updated_at"`
	}

	todo struct {
		ID        string  `json:"id"`
		Title     string  `json:"title"`
		Completed    bool  `json:"completed"`
		CreatedAt string  `json:"created_at"`
		UpdatedAt string  `json:"updated_at"`
	}
)

func init() {
	rnd = renderer.New()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use mongo.Connect instead of mongo.NewClient
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(hostName))
	checkErr(err, "MongoDB connection failed")

	// Ping the database to verify connection
	err = client.Ping(ctx, nil)
	checkErr(err, "MongoDB ping failed")

	// Select the database
	db = client.Database(dbName)

	log.Println("MongoDB connected!")
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	err := rnd.Template(w, http.StatusOK, []string{"static/index.tpl"}, nil)
	checkErr(err, "Template err")
}

func fetchTodos(w http.ResponseWriter, r *http.Request) {
	var todos []todoModel
	collection := db.Collection(collName)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		rnd.JSON(w, http.StatusInternalServerError, renderer.M{
			"message": "Failed to fetch todo lists",
			"error":   err,
		})
		return
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var t todoModel
		if err := cursor.Decode(&t); err != nil {
			rnd.JSON(w, http.StatusInternalServerError, renderer.M{
				"message": "Failed to decode todo",
				"error":   err,
			})
			return
		}
		todos = append(todos, t)
	}

	todoList := []todo{}
	for _, t := range todos {
		todoList = append(todoList, todo{
			ID:        t.ID.Hex(),
			Title:     t.Title,
			Completed:    t.Completed,
			CreatedAt: t.CreatedAt.Format(time.RFC3339),
			UpdatedAt: t.UpdatedAt.Format(time.RFC3339),
		})
	}

	rnd.JSON(w, http.StatusOK, renderer.M{
		"data": todoList,
	})
}

func createTodo(w http.ResponseWriter, r *http.Request) {
	var t todo
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message": "Failed to create todo",
			"error":   err,
		})
		return
	}

	if t.Title == "" {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "Failed to create todo",
			"error":   "Title is required",
		})
		return
	}

	tm := todoModel{
		ID:        primitive.NewObjectID(),
		Title:     t.Title,
		Completed:    t.Completed,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	collection := db.Collection(collName)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := collection.InsertOne(ctx, tm)
	if err != nil {
		rnd.JSON(w, http.StatusInternalServerError, renderer.M{
			"message": "Failed to create todo",
			"error":   err.Error(),
		})
		return
	}

	rnd.JSON(w, http.StatusCreated, renderer.M{
		"message": "Todo created successfully",
		"data":    tm,
	})
}

func updateTodo(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "Invalid id",
		})
		return
	}

	var t todo
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message": "Failed to update todo",
			"error":   err,
		})
		return
	}

	if t.Title == "" {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "Failed to update todo",
			"error":   "Title is required",
		})
		return
	}

	collection := db.Collection(collName)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"title":      t.Title,
			"completed":     t.Completed,
			"updated_at": time.Now(),
		},
	}

	_, err = collection.UpdateByID(ctx, objID, update)
	if err != nil {
		rnd.JSON(w, http.StatusInternalServerError, renderer.M{
			"message": "Failed to update todo",
			"error":   err,
		})
		return
	}

	rnd.JSON(w, http.StatusOK, renderer.M{
		"message": "Todo updated successfully",
	})
}

func deleteTodo(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "Invalid id",
		})
		return
	}

	collection := db.Collection(collName)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = collection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message": "Failed to delete todo",
			"error":   err,
		})
		return
	}

	rnd.JSON(w, http.StatusOK, renderer.M{
		"message": "Todo deleted successfully",
	})
}

func main() {
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", homeHandler)
	r.Route("/todo", func(r chi.Router) {
		r.Get("/", fetchTodos)
		r.Post("/", createTodo)
		r.Put("/{id}", updateTodo)
		r.Delete("/{id}", deleteTodo)
	})

	srv := &http.Server{
		Addr: port,
		Handler: r,
		IdleTimeout: 60 * time.Second,
		ReadTimeout: 5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	go func(){
		log.Println("Listening on port ", port)
		err := srv.ListenAndServe()
		checkErr(err, "Listen and serve err")
	}()

	<- stopCh
	log.Println("Shutting down server......")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	srv.Shutdown(ctx)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		checkErr(err, "Server forced to shutdown")
	}

	log.Println("Server stopped gracefully!")

}


func checkErr(err error, message ...string){
	if err != nil {
		if len(message) > 0 {
			log.Fatalf("%s: %v", message[0], err)
		} else {
			log.Fatal(err)	
		}
	}
}
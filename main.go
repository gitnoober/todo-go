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
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var rnd *renderer.Render
var db *mgo.Database
const (
	hostName string = "localhost:27017"
	dbName string = "demo_todo"
	collName string = "todo"
	port string = ":9000"
)
// Define the Status type
type Status int

// Enum for the different statuses
const (
	Pending Status = iota
	InProgress
	Completed
)

// String method for Status type
func (s Status) String() string {
	return [...]string{"pending", "in_progress", "completed"}[s]
}

// MarshalJSON implements the json.Marshaler interface for Status
func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalJSON implements the json.Unmarshaler interface for Status
func (s *Status) UnmarshalJSON(data []byte) error {
	var statusString string
	if err := json.Unmarshal(data, &statusString); err != nil {
		return err
	}
	switch statusString {
	case "pending":
		*s = Pending
	case "in_progress":
		*s = InProgress
	case "completed":
		*s = Completed
	default:
		log.Println("invalid status", statusString)
	}
	return nil
}

type(
	todoModel struct {
		ID 	bson.ObjectId `bson:"_id,omitempty"`
		Title string `bson:"title"`
		Status Status `bson:"status"`
		CreatedAt time.Time `bson:"created_at"`
		UpdatedAt time.Time `bson:"updated_at"`
	}

	todo struct {
		ID string `json:"id"`
		Title string `json:"title"`
		Status Status `json:"status"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	}
)



func init(){
	rnd = renderer.New()
	sess, err := mgo.Dial(hostName)
	checkErr(err)
	sess.SetMode(mgo.Monotonic, true) // Writes and Reads in the same connection, switched to a secondary once a write is done, helps in distributing load, since using the same connection you can read your writes
	db = sess.DB(dbName)
}

func homeHandler(w http.ResponseWriter, r *http.Request){
	err := rnd.Template(w, http.StatusOK, []string{"static/index.tpl"}, nil)
	checkErr(err, "Template err")
}

func fetchTodos(w http.ResponseWriter, r *http.Request){
	todos := []todoModel{}
	if err := db.C(collName).Find(bson.M{}).All(&todos); err != nil {
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message": "Failed to fetch todo lists",
			"error": err,
		})
		return
	}

	todoList := []todo{}

	for _, t := range todos {
		todoList = append(todoList, todo{
			ID: t.ID.Hex(),
			Title: t.Title,
			Status: t.Status,
			CreatedAt: t.CreatedAt.Format(time.RFC3339),
			UpdatedAt: t.UpdatedAt.Format(time.RFC3339),
		})
	}
	rnd.JSON(w, http.StatusOK, renderer.M{
		"data": todoList,
	})
}

func createTodo(w http.ResponseWriter, r *http.Request){
	var t todo
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message": "Failed to create todo",
			"error": err,
		})
	}

	if t.Title == "" {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "Failed to create todo",
			"error": "Title is required",
		})
		return
	}

	// Check for valid status
	if t.Status != Pending && t.Status != InProgress && t.Status != Completed {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "Failed to create todo",
			"error": "Invalid status provided",
		})
		return
	}

	tm := todoModel{
		ID: bson.NewObjectId(),
		Title: t.Title,
		Status: t.Status,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Insert the new todo into the database
	if err := db.C(collName).Insert(tm); err != nil {
		rnd.JSON(w, http.StatusInternalServerError, renderer.M{
			"message": "Failed to create todo",
			"error": err.Error(),
		})
		return
	}

	rnd.JSON(w, http.StatusCreated, renderer.M{
		"message": "Todo created successfully",
		"data":    tm,
	})
}

func updateTodo(w http.ResponseWriter, r *http.Request){
	
}

func deleteTodo(w http.ResponseWriter, r *http.Request){
	id := strings.TrimSpace(chi.URLParam(r, "id"))

	if !bson.IsObjectIdHex(id) {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "Invalid id",
		})
		return
	}

	if err := db.C(collName).RemoveId(bson.ObjectIdHex(id)); err != nil {
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message": "Failed to delete todo",
			"error": err,
		})
		return
	}
	rnd.JSON(w, http.StatusOK, renderer.M{
		"message": "Todo deleted successfully",
	})
}

func main(){
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


// func todoHandlers(r chi.Router) http.Handler{
// 	rg := chi.NewRouter()
// 	rg.Group(func(r chi.Router) {
// 		r.Get("/", fetchTodos)
// 		r.Post("/", createTodo)
// 		r.Put("/{id}", updateTodo)
// 		r.Delete("/{id}", deleteTodo)
// 	})
// 	return rg
// }



func checkErr(err error, message ...string){
	if err != nil {
		if len(message) > 0 {
			log.Fatalf("%s: %v", message[0], err)
		} else {
			log.Fatal(err)	
		}
	}
}


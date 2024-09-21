# Todo List API

A very simple Todo List API built with Go, Chi router, and MongoDB. This project demonstrates how to implement a basic RESTful API for managing Todo items with Create, Read, Update, and Delete (CRUD) operations.

## Features

- **Create Todo:** Add new todo items.
- **Read Todo:** Fetch the list of all todos.
- **Update Todo:** Update an existing todo's title or status.
- **Delete Todo:** Remove a todo item from the database.
- **Status Enum:** Todo items have three statuses (`Pending`, `InProgress`, and `Completed`).

## Technologies Used

- **Go (Golang)**
- **MongoDB (using the official driver)**
- **Chi Router**
- **TheDevSaddam Renderer** (for templating and JSON responses)

## Project Structure

```bash
.
├── main.go                 # Application entry point
├── go.mod                  # Go module file
└── static/
    └── index.tpl           # Template for home page
```

MongoDB Configuration

Make sure MongoDB is installed and running on your machine. The default connection string used in the project is:

```
const (
    hostName string = "127.0.0.1:27017"
    dbName string = "demo_todo"
    collName string = "todo"
    port string = ":9000"
)
```

If you want to change the MongoDB connection string, update the hostName constant in the code.

API Endpoints

	•	GET /todo/: Fetch all todos.
	•	POST /todo/: Create a new todo.
	•	PUT /todo/{id}: Update a specific todo by ID.
	•	DELETE /todo/{id}: Delete a specific todo by ID.

Todo Item Structure

The todo model in the API looks like this:
```
{
  "id": "string",          // Todo ID (auto-generated)
  "title": "string",       // Title of the todo
  "status": "string",      // Todo status (pending, in_progress, completed)
  "created_at": "string",  // Creation timestamp
  "updated_at": "string"   // Last update timestamp
}
```

Run it
```
go mod tidy
go run main.go
```
The server will be running at http://localhost:9000.

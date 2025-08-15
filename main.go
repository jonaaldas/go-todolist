package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type TODO struct {
	Todo string `json:"todo"`
	ID   string `json:"id"`
}

type USER struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type TodoList struct {
	UserID string `json:"user_id"`
	Todos  []TODO `json:"todos"`
}

func extractUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("user")

		if header == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User header required"})
			c.Abort()
			return
		}

		var user USER
		err := json.Unmarshal([]byte(header), &user)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user header format"})
			c.Abort()
			return
		}

		c.Set("user", user)
		c.Next()
	}
}

func main() {
	opt, _ := redis.ParseURL("rediss://default:ATSVAAIjcDEyZTA3NTkyYWM0M2E0OTMxOGVhNjcyZGMyMWIyMDU4N3AxMA@selected-albacore-13461.upstash.io:6379")
	client := redis.NewClient(opt)

	ctx := context.Background()

	router := gin.Default()
	router.Static("/assets", "./frontend/dist/assets")
	router.StaticFile("/", "./frontend/dist/index.html")
	router.GET("/ping", func(c *gin.Context) { c.JSON(200, gin.H{"message": "pong"}) })

	router.Use(extractUser())

	router.POST("/api/create", func(c *gin.Context) {
		user := c.MustGet("user").(USER)
		var todo TODO

		err := c.ShouldBind(&todo)
		todo.ID = uuid.New().String()

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var todoList TodoList // todoList = {userId: '', todos: []}
		existingData, redisErr := client.Get(ctx, "todos:"+user.Id).Result()

		if redisErr == redis.Nil {
			todoList = TodoList{
				UserID: user.Id,
				Todos:  []TODO{},
			}
		} else if redisErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Redis error"})
			return
		} else {
			// Parse existing todos
			if err := json.Unmarshal([]byte(existingData), &todoList); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse todos"})
				return
			}
		}

		todoList.Todos = append(todoList.Todos, todo)
		todoJSON, _ := json.Marshal(todoList)

		// now we save the todo list into Redis
		redisInsertError := client.Set(ctx, "todos:"+user.Id, todoJSON, 0).Err()

		if redisInsertError != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Redis Insert Error"})
			return
		}

		c.JSON(
			http.StatusOK, gin.H{
				"success": true,
			},
		)

	})

	router.GET("/api/all", func(c *gin.Context) {
		user := c.MustGet("user").(USER)
		allTodos, err := getTodos(client, ctx, "todos:"+user.Id)
		if err == redis.Nil {
			c.JSON(http.StatusOK, gin.H{
				"data":    []TODO{},
				"message": "No todos found",
			})
			return
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Database error",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": allTodos.Todos,
		})
	})

	router.DELETE("/api/delete/:id", func(c *gin.Context) {
		id := c.Param("id")
		fmt.Println("Deleting todo", id)
		user := c.MustGet("user").(USER)
		todoList, redisErrors := getTodos(client, ctx, "todos:"+user.Id)

		if redisErrors == redis.Nil {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "No todos found",
			})
			return
		} else if redisErrors != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Redis error",
			})
			return
		}

		var filteredTodos []TODO
		found := false
		for _, todo := range todoList.Todos {
			if todo.ID != id {
				filteredTodos = append(filteredTodos, todo)
			} else {
				found = true
			}
		}

		if !found {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Todo not found",
			})
			return
		}

		todoList.Todos = filteredTodos
		updatedJSON, _ := json.Marshal(todoList)

		if saveErr := client.Set(ctx, "todos:"+user.Id, updatedJSON, 0).Err(); saveErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to save updated todos",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Todo deleted successfully",
		})
	})

	router.DELETE("/api/delete/all", func(c *gin.Context) {
		user := c.MustGet("user").(USER)
		client.Del(ctx, "todos:"+user.Id)
		client.Del(ctx, "todos:"+user.Id)
		c.JSON(http.StatusOK, gin.H{
			"success": true,
		})
	})

	router.PUT("/api/edit/:id", func(c *gin.Context) {
		user := c.MustGet("user").(USER)
		id := c.Param("id")

		var body struct {
			NewUpdatedTodo string `json:"newUpdatedTodo"`
		}

		if err := c.ShouldBind(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}

		allTodos, err := getTodos(client, ctx, "todos:"+user.Id)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}

		found := false
		for i, todo := range allTodos.Todos {
			if todo.ID == id {
				allTodos.Todos[i] = TODO{
					ID:   id,
					Todo: body.NewUpdatedTodo,
				}
				found = true
				break
			}
		}

		if !found {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "Todo not found",
			})
			return
		}

		updatedJSON, _ := json.Marshal(allTodos)
		if err := client.Set(ctx, "todos:"+user.Id, updatedJSON, 0).Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Failed to save updated todos",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Todo edited successfully",
		})
	})

	router.Run()
}

func getTodos(client *redis.Client, ctx context.Context, key string) (TodoList, error) {
	var allTodos TodoList
	allJSONTodos, redisError := client.Get(ctx, key).Result()

	if redisError != nil {
		return allTodos, redisError
	}

	if unmarshalErr := json.Unmarshal([]byte(allJSONTodos), &allTodos); unmarshalErr != nil {
		return allTodos, unmarshalErr
	}

	if allTodos.Todos == nil {
		allTodos.Todos = []TODO{}
	}

	return allTodos, nil
}

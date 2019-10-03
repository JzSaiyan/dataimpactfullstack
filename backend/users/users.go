package users

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"sync"

	"../mongodb"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

// Users struct contains informations about
// a user
type Users struct {
	ID         string   `json:"id"`
	IsActive   bool     `json:"isActive"`
	Name       string   `json:"name"`
	Age        int      `json:"age"`
	Address    string   `json:"address"`
	Gender     string   `json:"gender"`
	Company    string   `json:"company"`
	Email      string   `json:"email"`
	Balance    string   `json:"balance"`
	Password   string   `json:"password"`
	About      string   `json:"about"`
	Registered string   `json:"registered"`
	Latitude   float64  `json:"latitude"`
	Longitude  float64  `json:"longitude"`
	Tags       []string `json:"tags"`
	Friends    []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	Data string `json:"data"`
}

// InitUsersRoutes initialises the users routes
func InitUsersRoutes(router *gin.Engine) {
	// GET endpoints
	router.GET("/user/:id", GetUsersHandler)
	router.GET("/users/list", GetUsersListHandler)

	// POST endpoints
	router.POST("/add/users", AddUsersHandler)

	// PUT endpoints
	router.PUT("/user/:id", UpdateUserHandler)

	// DELETE endpoints
	router.DELETE("/delete/user/:id", DeleteUserHandler)
}

// AddUsersHandler parses a file and store the users
// into the db and create a file with the user's data
func AddUsersHandler(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": http.StatusText(http.StatusInternalServerError)})
		return
	}

	files := form.File["file"]

	var wg sync.WaitGroup
	wg.Add(len(files))

	var users []Users
	queue := make(chan []Users, 1)

	errc := make(chan error)
	done := make(chan bool, 1)

	for _, file := range files {
		go func(nf *multipart.FileHeader) {
			f, err := nf.Open()
			if err != nil {
				errc <- err
				return
			}

			defer f.Close()
			data, err := ioutil.ReadAll(f)
			if err != nil {
				errc <- err
				return
			}

			var u []Users
			err = json.Unmarshal(data, &u)
			if err != nil {
				errc <- err
				return
			}

			queue <- u
		}(file)
	}

	go func() {
		for u := range queue {
			users = append(users, u...)
			wg.Done()
		}
	}()

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case err := <-errc:
		if err != nil {
			fmt.Println(err)
			c.JSON(http.StatusBadRequest, gin.H{"error": http.StatusText(http.StatusBadRequest)})
			return
		}
	}

	if err := addUsersToDB(&users); err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": http.StatusText(http.StatusBadRequest)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": "User(s) added"})
}

func addUsersToDB(users *[]Users) error {
	var wg sync.WaitGroup
	wg.Add(len(*users))

	errc := make(chan error)
	done := make(chan bool, 1)

	client, err := mongodb.GetMongoDBClient()
	if err != nil {
		return err
	}

	collection := client.Database("dataimpact").Collection("users")

	for _, user := range *users {
		go func(u Users) {
			defer wg.Done()
			hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
			if err != nil {
				errc <- err
				return
			}

			u.Password = string(hash)
			insertResult, err := collection.InsertOne(context.TODO(), u)
			if err != nil {
				errc <- err
				return
			}

			name := insertResult.InsertedID.(primitive.ObjectID).Hex()
			err = ioutil.WriteFile("./data/"+name, []byte(u.Data), os.ModePerm)
			if err != nil {
				errc <- err
				return
			}

		}(user)
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case err := <-errc:
		if err != nil {
			return err
		}
	}

	return nil
}

// DeleteUserHandler deletes a user with his ID
func DeleteUserHandler(c *gin.Context) {
	id := c.Param("id")

	client, err := mongodb.GetMongoDBClient()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": http.StatusText(http.StatusBadRequest)})
		return
	}

	collection := client.Database("dataimpact").Collection("users")
	filter := bson.M{"id": id}

	_, err = collection.DeleteOne(context.TODO(), filter)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": http.StatusText(http.StatusBadRequest)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": "deleted"})
}

// GetUsersListHandler gets the user list
func GetUsersListHandler(c *gin.Context) {
	var results []*Users

	client, err := mongodb.GetMongoDBClient()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": http.StatusText(http.StatusBadRequest)})
		return
	}

	collection := client.Database("dataimpact").Collection("users")

	cur, err := collection.Find(context.TODO(), bson.D{{}})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": http.StatusText(http.StatusBadRequest)})
		return
	}

	for cur.Next(context.TODO()) {
		var elem Users
		err := cur.Decode(&elem)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": http.StatusText(http.StatusBadRequest)})
			return
		}

		results = append(results, &elem)
	}

	if err := cur.Err(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": http.StatusText(http.StatusBadRequest)})
		return
	}

	cur.Close(context.TODO())
	c.JSON(http.StatusOK, results)
}

// GetUsersHandler gets the users with his ID
func GetUsersHandler(c *gin.Context) {
	id := c.Param("id")

	client, err := mongodb.GetMongoDBClient()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": http.StatusText(http.StatusBadRequest)})
		return
	}

	collection := client.Database("dataimpact").Collection("users")
	filter := bson.M{"id": id}

	var result Users
	err = collection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// UpdateUserHandler updates user info with his ID
func UpdateUserHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"version": 5})
}

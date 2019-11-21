package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/nattawat.s/golang-cloud-firestore/models"
	"github.com/twinj/uuid"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/mitchellh/mapstructure"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type App struct {
	Router *mux.Router
	client *firestore.Client
	ctx    context.Context
}

func main() {
	godotenv.Load()
	route := App{}
	route.Init()
	route.Run()
}

func (route *App) Init() {

	route.ctx = context.Background()

	sa := option.WithCredentialsFile("serviceAccountKey.json")
	app, err := firebase.NewApp(route.ctx, nil, sa)
	if err != nil {
		log.Fatalln(err)
	}

	route.client, err = app.Firestore(route.ctx)
	if err != nil {
		log.Fatalln(err)
	}
	route.Router = mux.NewRouter()
	route.initializeRoutes()
	fmt.Println("Successfully connected at port : " + route.GetPort())
}

func (route *App) GetPort() string {
	var port = os.Getenv("MyPort")
	if port == "" {
		port = "5000"
	}
	return ":" + port
}

func (route *App) Run() {
	log.Fatal(http.ListenAndServe(route.GetPort(), route.Router))
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func (route *App) initializeRoutes() {
	route.Router.HandleFunc("/", route.Home).Methods("GET")
	route.Router.HandleFunc("/books/{id}", route.FetchDataByID).Methods("GET")
	route.Router.HandleFunc("/create", route.CreateBook).Methods("POST")
	route.Router.HandleFunc("/books/{id}", route.EditDataByID).Methods("PUT")
	route.Router.HandleFunc("/books/{id}", route.DeleteDataByID).Methods("DELETE")
}

func (route *App) Home(w http.ResponseWriter, r *http.Request) {
	BooksData := []models.Books{}

	iter := route.client.Collection("books").Documents(route.ctx)
	for {
		BookData := models.Books{}
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			respondWithJSON(w, http.StatusInternalServerError, "Something wrong, please try again.")
		}

		mapstructure.Decode(doc.Data(), &BookData)
		BooksData = append(BooksData, BookData)
	}
	respondWithJSON(w, http.StatusOK, BooksData)
}

func (route *App) FetchDataByID(w http.ResponseWriter, r *http.Request) {

	params := mux.Vars(r)
	paramsID := params["id"]
	BooksData := []models.Books{}

	iter := route.client.Collection("books").Where("name", "==", paramsID).Documents(route.ctx)
	for {
		BookData := models.Books{}
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			respondWithJSON(w, http.StatusInternalServerError, "Something wrong, please try again.")
		}

		mapstructure.Decode(doc.Data(), &BookData)
		BooksData = append(BooksData, BookData)
	}

	respondWithJSON(w, http.StatusOK, BooksData)
}

func (route *App) CreateBook(w http.ResponseWriter, r *http.Request) {
	uid := uuid.NewV4()
	splitID := strings.Split(uid.String(), "-")
	id := splitID[0] + splitID[1] + splitID[2] + splitID[3] + splitID[4]

	BookData := models.Books{}

	Decoder := json.NewDecoder(r.Body)
	err := Decoder.Decode(&BookData)

	BookData.ID = id

	if err != nil {
		log.Printf("error: %s", err)
	}

	_, _, err = route.client.Collection("books").Add(route.ctx, BookData)
	if err != nil {
		log.Printf("An error has occurred: %s", err)
	}

	respondWithJSON(w, http.StatusCreated, "Create book success!")
}

func (route *App) EditDataByID(w http.ResponseWriter, r *http.Request) {

	params := mux.Vars(r)
	paramsID := params["id"]

	BookData := models.Books{}

	Decoder := json.NewDecoder(r.Body)
	err := Decoder.Decode(&BookData)
	if err != nil {
		log.Printf("error: %s", err)
	}

	var docID string

	BookData.ID = paramsID

	iter := route.client.Collection("books").Where("ID", "==", paramsID).Documents(route.ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			respondWithJSON(w, http.StatusInternalServerError, "Something wrong, please try again.")
		}
		docID = doc.Ref.ID
	}

	_, err = route.client.Collection("books").Doc(docID).Set(route.ctx, BookData)
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, "Something wrong, please try again.")
	}

	respondWithJSON(w, http.StatusCreated, "Edit book success!")
}

func (route *App) DeleteDataByID(w http.ResponseWriter, r *http.Request) {

	params := mux.Vars(r)
	paramsID := params["id"]

	var docID string

	iter := route.client.Collection("books").Where("ID", "==", paramsID).Documents(route.ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			respondWithJSON(w, http.StatusInternalServerError, "Something wrong, please try again.")
		}
		docID = doc.Ref.ID
	}

	_, err := route.client.Collection("books").Doc(docID).Delete(route.ctx)
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, "Something wrong, please try again.")
		return
	}

	respondWithJSON(w, http.StatusOK, "Delete book success !")
}

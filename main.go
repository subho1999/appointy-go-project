package main

import (
	//"fmt"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Participant ...
type Participant struct {
	name  string `json:"name" bson:"name"`
	email string `json:"email" bson:"email"`
	rsvp  string `json:"rsvp" bson:"rsvp"`
}

// Meeting ...
type Meeting struct {
	id                string        `json:"meeting_id" bson:"meeting_id"`
	title             string        `json:"title" bson:"title"`
	participants      []Participant `json:"participants" bson:"participants"`
	startTime         time.Time     `json:"start_time" bson:"start_time"`
	endTime           time.Time     `json:"end_time" bson:"end_time"`
	creationTimestamp time.Time     `json:"created" bson:"created"`
}

// InputStruct ...
type InputStruct struct {
	id           string        `json:"meeting_id" bson:"meeting_id"`
	title        string        `json:"title" bson:"title"`
	participants []Participant `json:"participants" bson:"participants"`
	startTime    string        `json:"start_time" bson:"start_time"`
	endTime      string        `json:"end_time" bson:"end_time"`
}

// Global database variables
var collection *mongo.Collection
var client *mongo.Client
var ctx context.Context

// Mutex variable to lock threads
var lock sync.Mutex

// Create Connection to MongoDB
func connectDB() {
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)

	defer cancel()

	err = client.Ping(context.Background(), readpref.Primary())
	if err != nil {
		log.Fatal("Couldn't connect to Database", err)
	} else {
		log.Println("Connected to DB!")
	}

	db := client.Database("appointyTestDB")
	collection = db.Collection("meetings")

	return
}

// Routes
func multiEndpointHandler(w http.ResponseWriter, r *http.Request) {

	lock.Lock()
	defer lock.Unlock()

	switch r.Method {
	case "GET":
		startTimeParam := r.URL.Query()["start"]
		endTimeParam := r.URL.Query()["end"]
		participantParam := r.URL.Query()["participant"]

		if len(startTimeParam) > 0 && len(endTimeParam) > 0 {

			startTime, err := stringToTime(startTimeParam[0])
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotAcceptable)
				return
			}

			endTime, err := stringToTime(endTimeParam[0])
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotAcceptable)
				return
			}

			cur, err := collection.Find(ctx, bson.D{})
			if err != nil {
				log.Fatal(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer cur.Close(ctx)

			var meetings []Meeting
			for cur.Next(ctx) {
				var meeting Meeting
				err = cur.Decode(&meeting)
				if err != nil {
					log.Fatal(err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				if startTime.Before(meeting.startTime) || startTime.Equal(meeting.startTime) &&
					endTime.After(meeting.endTime) || endTime.Equal(meeting.endTime) {
					meetings = append(meetings, meeting)
				}
			}

			js, err := json.Marshal(meetings)
			if err != nil {
				log.Printf("Error while marshalling JSON, Reason %v\n", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			w.Write(js)

		} else if len(participantParam) > 0 {

			participant := participantParam[0]
			cur, err := collection.Find(ctx, bson.D{})
			if err != nil {
				log.Fatal(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer cur.Close(ctx)

			var meetings []Meeting
			for cur.Next(ctx) {
				var meeting Meeting
				err = cur.Decode(&meeting)
				if err != nil {
					log.Fatal(err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				for _, p := range meeting.participants {
					if p.email == participant {
						meetings = append(meetings, meeting)
						break
					}
				}
			}

			js, err := json.Marshal(meetings)
			if err != nil {
				log.Printf("Error while marshalling JSON, Reason %v\n", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			w.Write(js)

		} else {
			log.Printf("Wrong GET Query called")
			http.Error(w, "Wrong GET Query called", http.StatusNotImplemented)
		}

	case "POST":

		var input InputStruct
		json.NewDecoder(r.Body).Decode(&input)

		var meeting Meeting
		meeting.id = input.id
		meeting.title = input.title
		meeting.participants = input.participants

		startTime, err := stringToTime(input.startTime)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotAcceptable)
			return
		}
		meeting.startTime = startTime

		endTime, err := stringToTime(input.endTime)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotAcceptable)
			return
		}
		meeting.endTime = endTime

		creationTime := time.Now()
		meeting.creationTimestamp = creationTime

		flag, index, err := checkInputValidity(meeting.participants, meeting.startTime, meeting.endTime)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if flag == true {
			log.Printf("Participant %s already has a meeting in this time period", meeting.participants[index].name)
			http.Error(w, "Participant "+meeting.participants[index].name+" already has a meeting in this time period", http.StatusNotAcceptable)
			return
		}
		_, err = collection.InsertOne(ctx, meeting)
		if err != nil {
			log.Printf("Error while inserting data, Reason %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		js, err := json.Marshal(meeting)
		if err != nil {
			log.Printf("Error while marshalling JSON, Reason %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)

	default:
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "Can't find method requested"}`))
	}
	// time.Sleep(2*time.Second)
}

func searchMeetingEndpoint(w http.ResponseWriter, r *http.Request) {
	meetingID := r.URL.Path[len("/meeting/"):]

	var meeting Meeting
	filter := bson.M{"meeting_id": meetingID}

	err := collection.FindOne(ctx, filter).Decode(&meeting)
	if err != nil {
		log.Printf("Error while retrieving data, Reason %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	js, err := json.Marshal(meeting)
	if err != nil {
		log.Printf("Error while marshalling JSON, Reason %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)

}

// Create Time object from String
func stringToTime(timeString string) (time.Time, error) {
	layout := "02-01-2006 03:04:05 PM" // Time input in JSON as DD-MM-YYYY HH:MM:SS AM
	t, err := time.Parse(layout, timeString)
	if err != nil {
		log.Printf("Error while parsing time, Reason %v\n", err.Error())
		return time.Now(), err
	}
	return t, nil
}

// Check for valid meeting details
func checkInputValidity(participants []Participant, meetingStart time.Time, meetingEnd time.Time) (bool, int, error) {
	var flag bool
	var err error
	for i, p := range participants {
		flag, err = checkParticipantAvailability(p.email, meetingStart, meetingEnd)
		if err != nil {
			return false, -1, err
		}
		if flag == true {
			return true, i, nil
		}
	}
	return false, -1, nil
}

// Check if participant already has meeting
func checkParticipantAvailability(email string, meetingStart time.Time, meetingEnd time.Time) (bool, error) {
	cur, err := collection.Find(ctx, bson.D{})
	if err != nil {
		log.Fatal(err)
		return false, err
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		var meeting Meeting
		err = cur.Decode(&meeting)
		if err != nil {
			log.Fatal(err)
			return false, err
		}

		for _, p := range meeting.participants {
			if p.email == email {
				if meetingStart.Before(meeting.endTime) && meetingStart.After(meeting.startTime) ||
					meetingEnd.After(meeting.startTime) && meetingEnd.Before(meeting.endTime) {
					if p.rsvp == "yes" {
						return true, nil
					}
				}
			}
		}
	}
	return false, nil
}

// Driver Function
func main() {
	connectDB()
	http.HandleFunc("/meetings", multiEndpointHandler)
	http.HandleFunc("/meeting/", searchMeetingEndpoint)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

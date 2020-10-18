package main

import (
	//"fmt"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Participant ...
type Participant struct {
	Name  string `json:"name" bson:"name"`
	Email string `json:"email" bson:"email"`
	RSVP  string `json:"rsvp" bson:"rsvp"`
}

// Meeting ...
type Meeting struct {
	ID                string        `json:"meeting_id" bson:"meeting_id"`
	Title             string        `json:"title" bson:"title"`
	Participants      []Participant `json:"participants" bson:"participants"`
	StartTime         time.Time     `json:"start_time" bson:"start_time"`
	EndTime           time.Time     `json:"end_time" bson:"end_time"`
	CreationTimestamp time.Time     `json:"created" bson:"created"`
}

// InputStruct ...
type InputStruct struct {
	ID           string        `json:"meeting_id" bson:"meeting_id"`
	Title        string        `json:"title" bson:"title"`
	Participants []Participant `json:"participants" bson:"participants"`
	StartTime    string        `json:"start_time" bson:"start_time"`
	EndTime      string        `json:"end_time" bson:"end_time"`
}

// Global database variables
var collection *mongo.Collection
var client *mongo.Client
var ctx context.Context

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

				if startTime.Before(meeting.StartTime) || startTime.Equal(meeting.StartTime) &&
					endTime.After(meeting.EndTime) || endTime.Equal(meeting.EndTime) {
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

				for _, p := range meeting.Participants {
					if p.Email == participant {
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
		meeting.ID = input.ID
		meeting.Title = input.Title
		meeting.Participants = input.Participants

		startTime, err := stringToTime(input.StartTime)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotAcceptable)
			return
		}
		meeting.StartTime = startTime

		endTime, err := stringToTime(input.EndTime)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotAcceptable)
			return
		}
		meeting.EndTime = endTime

		creationTime := time.Now()
		meeting.CreationTimestamp = creationTime

		flag, index, err := checkInputValidity(meeting.Participants, meeting.StartTime, meeting.EndTime)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if flag == true {
			log.Printf("Participant %s already has a meeting in this time period", meeting.Participants[index].Name)
			http.Error(w, "Participant "+meeting.Participants[index].Name+" already has a meeting in this time period", http.StatusNotAcceptable)
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
		flag, err = checkParticipantAvailability(p.Email, meetingStart, meetingEnd)
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

		for _, p := range meeting.Participants {
			if p.Email == email {
				if meetingStart.Before(meeting.EndTime) && meetingStart.After(meeting.StartTime) ||
					meetingEnd.After(meeting.StartTime) && meetingEnd.Before(meeting.EndTime) {
					if p.RSVP == "yes" {
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

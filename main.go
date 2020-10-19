package main

import (
	//"fmt"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
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

// ResponseStruct ...
type ResponseStruct struct {
	StartIndex    int       `json:"start_index" bson:"start_index"`
	EndIndex      int       `json:"end_index" bson:"end_index"`
	MeetingsArray []Meeting `json:"meetings" bson:"meetings"`
}

// Global database variables

// URI ...
var URI string = "mongodb://localhost:27017"
var collection *mongo.Collection
var ctx context.Context

// Layout ...
var layout string = "02-01-2006 03:04:05 PM" // Time input in JSON as DD-MM-YYYY HH:MM:SS AM

// Mutex Locks
var lock sync.Mutex

// Create Connection to MongoDB
func connectDB() {
	client, err := mongo.NewClient(options.Client().ApplyURI(URI))
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

// Routes for /meetings
func multiEndpointHandler(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case "GET":

		startTimeParam := r.URL.Query()["start"]
		endTimeParam := r.URL.Query()["end"]
		participantParam := r.URL.Query()["participant"]

		if len(startTimeParam) > 0 && len(endTimeParam) > 0 {

			queryHandlerTimeDuration(w, r, startTimeParam[0], endTimeParam[0])

		} else if len(participantParam) > 0 {

			queryHandlerParticipantEmail(w, r, participantParam[0])

		} else {
			log.Printf("Wrong GET Query called")
			http.Error(w, "Wrong GET Query called", http.StatusNotImplemented)
		}

	case "POST":

		createNewMeetingHandler(w, r)

	default:
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"message": "Can't find method requested"}`))
		if err != nil {
			log.Printf("Error while writing message")
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// Routes for /meeting/:id
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
	_, err = w.Write(js)
	if err != nil {
		log.Printf("Error while writing message %v", js)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}

// Function to handle time range GET request
func queryHandlerTimeDuration(w http.ResponseWriter, r *http.Request, startTimeString string, endTimeString string) {
	startTime, err := stringToTime(startTimeString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotAcceptable)
		return
	}

	endTime, err := stringToTime(endTimeString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotAcceptable)
		return
	}

	cur, err := collection.Find(ctx, bson.D{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Fatal(err)
		return
	}
	defer cur.Close(ctx)

	var meetings []Meeting
	for cur.Next(ctx) {
		var meeting Meeting
		err = cur.Decode(&meeting)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Fatal(err)
			return
		}

		if startTime.Before(meeting.StartTime) || startTime.Equal(meeting.StartTime) &&
			endTime.After(meeting.EndTime) || endTime.Equal(meeting.EndTime) {
			meetings = append(meetings, meeting)
		}
	}

	var responseStruct ResponseStruct
	offsetParam := r.URL.Query()["offset"]
	if len(offsetParam) > 0 {
		offsetIndex, err := strconv.Atoi(offsetParam[0])
		if err != nil {
			log.Printf("Error converting offset string to integer, %v\n", err.Error())
			http.Error(w, "Offset value "+offsetParam[0]+" is not a valid integer", http.StatusBadRequest)
		}
		if (offsetIndex < 0) {
			log.Printf("Negative offset Parameter %d", offsetIndex)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var reducedMeetings []Meeting
		var i int
		for i = offsetIndex; i < len(meetings) && i < offsetIndex+10; i++ {
			reducedMeetings = append(reducedMeetings, meetings[i])
		}
		responseStruct.StartIndex = offsetIndex
		responseStruct.EndIndex = i
		responseStruct.MeetingsArray = reducedMeetings

	} else {
		responseStruct.StartIndex = 0
		responseStruct.EndIndex = len(meetings)
		responseStruct.MeetingsArray = meetings
	}

	js, err := json.Marshal(responseStruct)
	if err != nil {
		log.Printf("Error while marshalling JSON, Reason %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(js)
	if err != nil {
		log.Printf("Error while writing message %v", js)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Function to handle participant Email GET request
func queryHandlerParticipantEmail(w http.ResponseWriter, r *http.Request, participantEmail string) {
	cur, err := collection.Find(ctx, bson.D{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Fatal(err)
		return
	}
	defer cur.Close(ctx)

	var meetings []Meeting
	for cur.Next(ctx) {
		var meeting Meeting
		err = cur.Decode(&meeting)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Fatal(err)
			return
		}

		for _, p := range meeting.Participants {
			if p.Email == participantEmail {
				meetings = append(meetings, meeting)
				break
			}
		}
	}

	var responseStruct ResponseStruct
	offsetParam := r.URL.Query()["offset"]
	if len(offsetParam) > 0 {
		offsetIndex, err := strconv.Atoi(offsetParam[0])
		if err != nil {
			log.Printf("Error converting offset string to integer, %v\n", err.Error())
			http.Error(w, "Offset value "+offsetParam[0]+" is not a valid integer", http.StatusBadRequest)
			return
		}
		if (offsetIndex < 0) {
			log.Printf("Negative offset Parameter %d", offsetIndex)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var reducedMeetings []Meeting
		var i int
		for i = offsetIndex; i < len(meetings) && i < offsetIndex+10; i++ {
			reducedMeetings = append(reducedMeetings, meetings[i])
		}
		responseStruct.StartIndex = offsetIndex
		responseStruct.EndIndex = i
		responseStruct.MeetingsArray = reducedMeetings

	} else {
		responseStruct.StartIndex = 0
		responseStruct.EndIndex = len(meetings)
		responseStruct.MeetingsArray = meetings
	}

	js, err := json.Marshal(responseStruct)
	if err != nil {
		log.Printf("Error while marshalling JSON, Reason %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(js)
	if err != nil {
		log.Printf("Error while writing message %v", js)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Function to handle POST requests
func createNewMeetingHandler(w http.ResponseWriter, r *http.Request) {

	lock.Lock()
	defer lock.Unlock()

	var input InputStruct
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		log.Printf("Error while decoding input %v", err.Error())
		http.Error(w, "Error while decoding input "+err.Error(), http.StatusBadRequest)
		return
	}

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
	_, err = w.Write(js)
	if err != nil {
		log.Printf("Error while writing message %v", js)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Create Time object from String
func stringToTime(timeString string) (time.Time, error) {
	
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
		flag, err = checkParticipantAvailability(p.Email, p.RSVP, meetingStart, meetingEnd)
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
func checkParticipantAvailability(email string, currentRSVP string, meetingStart time.Time, meetingEnd time.Time) (bool, error) {
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
					if p.RSVP == "yes" && currentRSVP == "yes" {
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

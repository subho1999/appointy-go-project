package main

import(
	"testing"
	//"net/http"
	//"net/http/httptest"
	"context"
	"time"
	//"encoding/json"
	//"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Global Test Variables
var ctxTest context.Context
var collectionTest *mongo.Collection

// func TestSearchMeetingEndpoint(t *testing.T) {
// 	req, err := http.NewRequest(http.MethodGet, "/meeting/11216", nil)
// 	if err != nil {
// 		t.Fatal("Fatal Error 1", err)
// 	}
// 	rr := httptest.NewRecorder()
	
// 	searchMeetingEndpoint(rr, req)

// 	if rr.Code != http.StatusOK {
// 		t.Errorf("Handler returned wrong status code: got %v expected %v", rr.Code, http.StatusOK)
// 	}
// }

// func TestPostHandler(t *testing.T) {
// 	postData := `{"meeting_id": "681351","title": "OOPS Class","participants": [{"name": "P1","email": "e1@gmail.com","rsvp": "yes"},{"name": "P2","email": "e2@gmail.com","rsvp": "no"},{"name": "P4","email": "e4@gmail.com","rsvp": "maybe"}],"start_time": "20-10-2020 11:00:00 AM","end_time": "20-10-2020 12:30:00 PM"}`

// 	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		w.WriteHeader(http.StatusOK)
// 		if r.Method != "POST" {
// 			t.Errorf("Expected POST request, got %s request", r.Method)
// 		}
// 		if r.URL.EscapedPath() != "/meetings" {
// 			t.Errorf("Expected request to /meetings, got %s", r.URL.EscapedPath)
// 		}

// 		var input InputStruct
// 		err := json.NewDecoder(r.Body).Decode(&input)
// 		if err != nil {
// 			t.Fatal(err)
// 		}
		


// 	}))
// 	resp, err := http.Post()
// }

func TestMain(t *testing.T) {
	setup(t)

}

func setup(t *testing.T) {
	client, err := mongo.NewClient(options.Client().ApplyURI(URI))
	if err != nil {
		t.Fatal(err.Error())
	}

	ctxTest, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctxTest)

	defer cancel()

	err = client.Ping(context.Background(), readpref.Primary())
	if err != nil {
		t.Fatal("Couldn't connect to Database", err)
	} else {
		t.Log("Connected to DB!")
	}

	dbTest := client.Database("testDB")
	collectionTest = dbTest.Collection("meetingsTest")

	p1 := Participant{
		Name: "P1",
		Email: "e1@gmail.com",
		RSVP: "yes",
	}
	p2 := Participant{
		Name: "P2",
		Email: "e2@gmail.com",
		RSVP: "no",
	}
	p3 := Participant{
		Name: "P3",
		Email: "e3@gmail.com",
		RSVP: "yes",
	}
	st1, _ := time.Parse(layout, "18-10-2020 09:00:00 AM")
	et1, _ := time.Parse(layout, "18-10-2020 11:00:00 AM")
	ct1, _ := time.Parse(layout, "18-10-2020 05:30:00 AM")
	m1 := Meeting {
		ID: "11216",
		Title: "Meeting 1",
		Participants: []Participant{p1, p2},
		StartTime: st1,
		EndTime: et1,
		CreationTimestamp: ct1,
	}
	st2, _ := time.Parse(layout, "19-10-2020 09:00:00 AM")
	et2, _ := time.Parse(layout, "19-10-2020 11:00:00 AM")
	ct2, _ := time.Parse(layout, "19-10-2020 05:30:00 AM")
	m2 := Meeting {
		ID: "14345",
		Title: "Meeting 2",
		Participants: []Participant{p1, p3},
		StartTime: st2,
		EndTime: et2,
		CreationTimestamp: ct2,
	}
	_, err = collectionTest.InsertOne(ctxTest, m1)
	if err != nil {
		t.Fatal(err)
	}
	_, err = collectionTest.InsertOne(ctxTest, m2)
	if err != nil {
		t.Fatal(err)
	}
}
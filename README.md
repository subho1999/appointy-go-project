Download mongo drivers by go get go.mongodb.org/mongo-driver/mongo

Build main.go using: go build main.go

The API endpoints are:

1.  GET /meeting/<id>
2.  GET /meetings?start=<start-time>&end=<end-time>
3.  GET /meetings?participant=<email>
4.  POST /meetings

The format of JSON that is passed in POST body is:

```
{
    "meeting_id": string
    "title": string
    "participants": [
        {
            "name": string,
            "email": string,
            "rsvp": string
        },
        ...
    ],
    "start_time": string
    "end_time": string
}
```

The start_time and end_time values must be of the format: "DD-MM-YYYY HH:MM:SS AM"

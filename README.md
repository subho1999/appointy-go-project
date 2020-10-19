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
    "meeting_id": string,
    "title": string,
    "participants": [
        {
            "name": string,
            "email": string,
            "rsvp": string
        },
        ...
    ],
    "start_time": string,
    "end_time": string
}
```

The start_time and end_time values must be of the format: "DD-MM-YYYY HH:MM:SS AM"

For the GET requests the API also provides offset pagination.
For offset pagination, the URL Query Parameter [I]offset[/I] should be used with 0 indexing.
If pagination is used 10 entries at most are returned

The JSON structure of the response for GET /meetings?start=<start-time>&end=<end-time>&offset=<offset> 
and GET /meetings?participant=<email>&offset=<offset> is:
```
{
    "start_index": int,
    "end_index": int,
    "meetings": [
        {
            meeting_id": string,
            "title": string,
            "participants": [
                {
                    "name": string,
                    "email": string,
                    "rsvp": string
                },
                ...
            ],
            "start_time": string,
            "end_time": string
        },
        ...
    ]
}
```

tcptap
======

A chat app written in Go.

The goal of this project was to explore what was required to support 
non-trivial chat applications with as simple a protocol as possible. 

### Data layer:
There are conversations and users. A user may be a participant in a 
conversation. Messages are attributed to the user that sent them (duh).
In a sql database, it might look like this:

              Participations
                //      \\
    Conversations        Users
                \\      //
                 Messages

### Client:
The client is a command line client. You can run it by executing the
following command:

    tcptap connTapClient <host:port>

The client is pretty awesome.

### Docker:
To run the server with docker:

    docker build -t seanpont/tcptap .
    docker run --rm -p 8080:8080 -it seanpont/tcptap

### Please note:
This is not a production app. It is not supremely well tested.
There are known bugs. It is an experiment.


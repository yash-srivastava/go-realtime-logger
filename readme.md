REAL-TIME LOGGING
-


https://yash-srivastava.github.io/go-realtime-logger/

Intro:
-
This Package Runs HTTP Server on the specified PORT to display the CONSOLE OUTPUT of the given PROCESS in REAL-TIME.

Requirements:
-
` $ go get github.com/boltdb/bolt/...`

` $ go get github.com/googollee/go-socket.io`

Usage:
-
` $ go build realtime_logger`

` $ ./realtime_logger [PROCESS] [PROCESS_ARGS] [PORT]`

Now open localhost:PORT in the browser

![Sample](img.jpg "Sample") 

Features:
-
- Real time console output on a web page
- Persistence of Logs over the Time
- Retrieval of Logs based on Timestamp

More Info:
- 
The use of BoltDB to store logs locally makes it thread-safe. It uses websocket to connect display logs in real-time.

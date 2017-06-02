package main
import (
	"os/exec"
	"log"
	"os"
	"bufio"
	"github.com/googollee/go-socket.io"
	"net/http"
	"time"
	"github.com/boltdb/bolt"
	"bytes"
	"encoding/json"
	"encoding/binary"
	"io"
	"flag"
)
var (
	logger					*log.Logger
	server					*socketio.Server
	so 					socketio.Socket
	buff 					[]string
	isconn					bool
	db					*bolt.DB
	cmdPath					string
	args					[]string
	port 					string
)

const (
	maxbufflength = 5000
)

type data struct {
	Time string
	Val  string
}

func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func get(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	loc, _ := time.LoadLocation("Asia/Kolkata")
	tf,_:=time.ParseInLocation("01/02/2006 3:04 PM",from,loc)
	tt,_:=time.ParseInLocation("01/02/2006 3:04 PM",to,loc)
	tf=tf.Add(-1*time.Minute)
	tt=tt.Add(1*time.Minute)
	logger.Println("from=",tf," to=",tt)
	restr:=""
	db.View(func(tx *bolt.Tx) error {
		// Assume our events bucket exists and has RFC3339 encoded time keys.
		c := tx.Bucket([]byte("Ind")).Cursor()
		min:=tf.Format(time.RFC3339)
		max:=tt.Format(time.RFC3339)
		logger.Println("From=",min," To=",max)
		_, v := c.Seek([]byte(min))
		mi:=v
		_, v = c.Seek([]byte(max))
		ma:=v
		if len(v)==0{
			_,ma=c.Last()
		}
		c = tx.Bucket([]byte("Log")).Cursor()
		x:=0
		for k, v := c.Seek((mi)); k != nil && bytes.Compare(k, (ma)) <= 0; k, v = c.Next() {
			if x!=0{
				restr+="##"
			}
			da:=data{}
			err:=json.Unmarshal(v,&da)
			if err!=nil{
				logger.Println(err)
			}
			t,_:=time.ParseInLocation(time.RFC3339,da.Time,loc)
			ss:=t.Format(time.RFC850)
			ss+="=>"+da.Val
			restr+=ss
			x++
		}
		return nil
	})
	io.WriteString(w,restr)
}
func updateDB(t time.Time,ss string){
	id:=uint64(0)
	eorr := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Log"))
		id, _ = b.NextSequence()
		da:=data{}
		da.Time=t.Format(time.RFC3339)
		da.Val=ss
		buf, err := json.Marshal(da)
		if err != nil {
			return err
		}
		err = b.Put(itob((id)), buf)
		return err
	})
	if eorr!=nil{
		logger.Println(eorr)
	}
	eorr = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Ind"))
		err := b.Put([]byte(t.Format(time.RFC3339)), itob(id))
		return err
	})
	if eorr!=nil{
		logger.Println(eorr)
	}
}
func runServer() {
	logger.Println("Starting Process!!!")

	cmd:=exec.Command(cmdPath,args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logger.Println(err)
	}
	// start the command after having set up the pipe
	if err := cmd.Start(); err != nil {
		logger.Println(err)
	}
	// read command's stdout line by line
	in := bufio.NewScanner(stdout)
			for in.Scan() {
				t:=time.Now()
				ss:=in.Text()
				go updateDB(t,ss)
				str:=t.Format(time.RFC850)+"=>"
				str+=ss
				logger.Println("out=",str) // write each line to your log, or anything you need
				if len(buff)>maxbufflength{
					buff=buff[1:]
				}
				buff=append(buff,str)
				if isconn{
					so.Emit("chat message", str)
					so.BroadcastTo("chat","chat message", str)
				}

			}
	if err := in.Err(); err != nil {
		logger.Println("error: %s", err)
	}

}

func main() {
	logger =log.New(os.Stdout, "REAL-TIME-LOGGER: ", log.Lshortfile)
	var err error
	flag.Parse()
	if len(flag.Args()) < 1 {
		log.Fatal("USAGE PROCESS_NAME ARGS PORT")
	}
	cmdPath, err = exec.LookPath(flag.Args()[0])
	if err != nil {
		log.Fatal(err)
	}
	if len(flag.Args())>2{
		args = flag.Args()[1:len(flag.Args())-1]
	}else{
		args = nil
	}

	port = flag.Args()[len(flag.Args())-1]

	db, err = bolt.Open("log.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	maidd:=uint64(0)
	eorr := db.Update(func(tx *bolt.Tx) error {
		//tx.DeleteBucket([]byte("Log"))
		//tx.DeleteBucket([]byte("Ind"))
		_, err := tx.CreateBucketIfNotExists([]byte("Log"))
		if err != nil {
			logger.Println("create bucket: ", err)
			return err
		}
		_, err = tx.CreateBucketIfNotExists([]byte("Ind"))
		if err != nil {
			logger.Println("create bucket: ", err)
			return err
		}
		return nil
	})
	if eorr!=nil{
		logger.Println(eorr)
	}
	go runServer()
	server, err = socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}
	loc, _ := time.LoadLocation("Asia/Kolkata")
	server.On("connection", func(so1 socketio.Socket) {
		so=so1
		so.Join("chat")
		isconn=true
		log.Println("on connection")
		restr:=""
		db.View(func(tx *bolt.Tx) error {
			// Assume our events bucket exists and has RFC3339 encoded time keys.
			c := tx.Bucket([]byte("Log")).Cursor()
			maid,_:=c.Last()
			if len(maid)!=0{
				maidd=binary.BigEndian.Uint64(maid)
			}else{
				maidd=uint64(0)
			}
			id:=(maidd)
			logger.Println("max=",id)
			min := uint64(0)
			if id>2000{
				min=id-2000
			}
			max := id
			x:=0
			for k, v := c.Seek(itob(min)); k != nil && bytes.Compare(k, itob(max)) <= 0; k, v = c.Next() {
				if x!=0{
					restr+="##"
				}
				da:=data{}
				err:=json.Unmarshal(v,&da)
				if err!=nil{
					logger.Println(err)
				}
				t,_:=time.ParseInLocation(time.RFC3339,da.Time,loc)
				ss:=t.Format(time.RFC850)
				ss+="=>"+da.Val
				restr+=ss
				x++
			}

			return nil
		})
		if restr!=""{
			so.Emit("chat message", restr)
		}

	})
	server.On("error", func(so socketio.Socket, err error) {
		log.Println("error:", err)
	})
	http.Handle("/socket.io/", server)
	http.Handle("/", http.FileServer(http.Dir("./src/realtime_logger/asset")))
	http.HandleFunc("/get", get)
	log.Println("Serving at :"+port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

package main

import (
	"os"
	"log"
	"fmt"
	"bytes"
	"sync"
	"flag"
	"context"
	"encoding/base64"
	
	"github.com/jackc/pgx/v5"
	"github.com/schollz/progressbar/v3"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var (
	psqlUrl string
	s3Url string
	s3Key string
	s3Secret string
	s3Bucket string
	offset int
	limit int
	debug bool
	s3Client *minio.Client
)

type NodestoreNode struct {
	ID string
	Data string
}

func init() {
	flag.StringVar(&psqlUrl, "db", "postgres://postgres:@localhost:5432", "postgres db url. e.g. postgres://postgres:@localhost:5432")
	flag.StringVar(&s3Url, "s3domain", "", "s3 provider domain eg. s3.example.com")
	flag.StringVar(&s3Key, "s3key", "", "s3 access key.")
	flag.StringVar(&s3Secret, "s3secret", "", "s3 secret")
	flag.StringVar(&s3Bucket, "s3bucket", "", "s3 bucket name")
	flag.IntVar(&offset, "offset", 0, "offset to start at")
	flag.IntVar(&limit, "limit", 1000, "max amount of rows to parse at once")
	flag.BoolVar(&debug, "debug", false, "debug mode shows more infomation")
	flag.Parse()

	if s3Url == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if s3Key == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}
	
	if s3Secret == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}
	
	if s3Bucket == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}
}

func main() {
	asd, err := minio.New(s3Url, &minio.Options{
		Creds:  credentials.NewStaticV4(s3Key, s3Secret, ""),
		Secure: true,
	})
	if err != nil {
		panic(err)
	}
	s3Client = asd

	dbPool, err := pgx.Connect(context.Background(), psqlUrl)
	if err != nil {
		panic(err)
	}
	defer dbPool.Close(context.Background())
	
	run := true
	for run {
		log.Printf("Migrating next %d entries. Current offset: %d", limit, offset)

		nodes := getRows(dbPool, offset)

		if len(nodes) == 0 {
			fmt.Fprintf(os.Stderr, "No more rows. Finished!")
			run = false
		}

		var wg sync.WaitGroup
		bar := progressbar.Default(int64(limit))
		for _, node := range nodes {
			wg.Add(1)

			go func() {
				defer wg.Done()
				defer bar.Add(1)
				processRow(&node)
			}()
		}
		wg.Wait()

		offset += limit
	} 
}

func getRows(dbPool *pgx.Conn, offset int) (nodes []NodestoreNode) {
	rows, err := dbPool.Query(context.Background(), "SELECT id, data FROM public.nodestore_node ORDER BY timestamp DESC LIMIT $1 OFFSET $2", limit, offset)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Query failed: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	for rows.Next() {
		var node NodestoreNode

		err := rows.Scan(&node.ID, &node.Data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Next failed: %v\n", err)
			os.Exit(1)
		}

		nodes = append(nodes, node)
	}
	if err = rows.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Next failed: %v\n", err)
		os.Exit(1)
	}

	return
}

func processRow(node *NodestoreNode) {
	sDec, err := base64.StdEncoding.DecodeString(node.Data)
	if err != nil {
		fmt.Println(err)
		return
	}

	r := bytes.NewReader(sDec)

	_, err = s3Client.PutObject(context.Background(), s3Bucket, node.ID, r, r.Size(), minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		fmt.Println("Failed to upload: ", err)
		return
	}

	if debug {
		log.Printf("Done running node id %s", node.ID)
	}
}

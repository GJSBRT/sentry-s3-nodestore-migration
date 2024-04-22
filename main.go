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
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var (
	psqlUrl string
	s3Url string
	s3Key string
	s3Secret string
	s3Bucket string
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

	if s3Url == "" {
		panic("need an s3 domain. Provider one with the --s3domain parameter")
	}

	if s3Key == "" {
		panic("need an s3 key. Provider one with the --s3key parameter")
	}
	
	if s3Secret == "" {
		panic("need an s3 secret. Provider one with the --s3secret parameter")
	}
	
	if s3Bucket == "" {
		panic("need an s3 bucket name. Provider one with the --s3bucket parameter")
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
	var offset int
	for run {
		log.Printf("offset %d", offset)

		nodes := getRows(dbPool, offset)

		if len(nodes) == 0 {
			fmt.Fprintf(os.Stderr, "No more rows. Finished!")
			run = false
		}

		var wg sync.WaitGroup
		for _, node := range nodes {
			wg.Add(1)

			go func() {
				defer wg.Done()
				processRow(&node)
			}()
		}
		wg.Wait()

		offset += 1000
	} 
}

func getRows(dbPool *pgx.Conn, offset int) (nodes []NodestoreNode) {
	rows, err := dbPool.Query(context.Background(), "SELECT id, data FROM public.nodestore_node LIMIT 1000 OFFSET $1", offset)
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

	log.Printf("Done running node id %s", node.ID)
}

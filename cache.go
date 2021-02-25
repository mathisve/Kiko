package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func getS3() (svc *s3.S3, err error) {
	sess, err := session.NewSession(&aws.Config{
		Region: &config.Backend.Config.Region,
	})
	if err != nil {
		return svc, err
	}

	svc = s3.New(sess)
	return svc, err
}

func GetCache() (err error) {
	if useLocalFile {
		if fileExists(HashcacheFilename) {
			b, err := readFile(HashcacheFilename)
			if err != nil {
				return err
			}

			err = json.Unmarshal(b, &hashCache)
			if err != nil {
				return err
			}
		}

		return err
	}

	svc, err := getS3()
	if err != nil {
		return err
	}

	result, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: &config.Backend.Config.Bucket,
		Key:    aws.String(fmt.Sprintf("%v/%v", config.Backend.Config.Key, HashcacheFilename)),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey:
				return errors.New(ErrFileDoesNotExist)
			case s3.ErrCodeInvalidObjectState:
				return errors.New(s3.ErrCodeInvalidObjectState)
			default:
				return aerr
			}
		} else {
			return aerr
		}
	}

	// unmarshal content into hashCache
	b, err := ioutil.ReadAll(result.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(b, &hashCache)
	if err != nil {
		return err
	}

	return err
}

func SaveCache() (err error) {
	f, err := json.MarshalIndent(newHashCache, "", "	")
	if err != nil {
		return err
	}

	if useLocalFile {
		err = os.Remove(HashcacheFilename)
		if err != nil {
			return err
		}

		err = os.WriteFile(HashcacheFilename, f, 0644)
		if err != nil {
			return err
		}

		return err
	}

	svc, err := getS3()
	if err != nil {
		return err
	}

	_, err = svc.PutObject(&s3.PutObjectInput{
		Body:   bytes.NewReader(f),
		Bucket: &config.Backend.Config.Bucket,
		Key:    aws.String(fmt.Sprintf("%v/%v", config.Backend.Config.Key, HashcacheFilename)),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				return aerr
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			return aerr
		}
	}

	return err
}

func fileExists(filename string) bool {
	if _, err := os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func readFile(filename string) (b []byte, err error) {
	info, err := os.Stat(filename)
	if err != nil {
		return b, err
	}

	b = make([]byte, info.Size())
	f, err := os.Open(filename)
	if err != nil {
		return b, err
	}

	_, err = f.Read(b)
	if err != nil {
		return b, err
	}

	return b, err
}

func hashBytes(b []byte) []byte {
	h := sha256.New()
	h.Write(b)
	return h.Sum(nil)
}

func getHashFromCache(path string) ([]byte, error) {
	for _, hash := range hashCache.Cache {
		if hash.Path == path {
			return hash.Hash, nil
		}
	}

	return *new([]byte), errors.New(ErrPathNotFound)
}

func (cache HashCache) appendToCache(d Cache) {
	hashCacheLock.Lock()
	defer hashCacheLock.Unlock()

	newHashCache.Cache = append(newHashCache.Cache, d)
}

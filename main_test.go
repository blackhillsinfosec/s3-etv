package main

import (
    "os"
    "testing"
)

var (
    testS3Region    string
    testS3ObjectUrl string
    testInputFile   string
)

func init() {
    testS3Region = os.Getenv("TEST_AWS_REGION")
    if testS3Region == "" {
        testS3Region = defaultS3Region
        iLog.Printf("Using default AWS region: %v", testS3Region)
    }
    testS3ObjectUrl = os.Getenv("TEST_S3_OBJECT_URL")
    if testS3ObjectUrl == "" {
        panic("TEST_S3_OBJECT_URL variable required for tests")
    }
    testInputFile = os.Getenv("TEST_INPUT_FILE")
    if testInputFile == "" {
        panic("TEST_INPUT_FILE variable required for tests")
    }
}

func TestPullETag(t *testing.T) {
    if etag, err := getS3ObjectETag(testS3ObjectUrl, testS3Region); err != nil {
        t.Log("Failed to pull etag")
        t.Fatalf("%v", err)
    } else {
        t.Logf("Retrieved ETag for %s: %s", testS3ObjectUrl, etag.Value)
    }
}

func TestValidate(t *testing.T) {
    inputFile = testInputFile
    s3ObjectUrl = testS3ObjectUrl
    s3Region = testS3Region
    validate()
}

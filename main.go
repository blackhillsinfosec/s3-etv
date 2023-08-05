package main

import (
    "fmt"
    "math"
    "regexp"
    "time"
)

var (
    inputFile        string
    s3ObjectUrl      string
    s3Region         string
    processCount     int
    defaultS3Region  = "us-east-1"
    md5Reg, _        = regexp.Compile("(?i)([0-9|A-F]){32}")
    mB               = 1024 * 1024
    defaultChunkSize = mB * 5
)

func main() {
    rootCmd.Execute()
}

func validate() {

    //===============
    // PARSE THE ETAG
    //===============

    var etag *eTag
    var err error
    if etag, err = getS3ObjectETag(s3ObjectUrl, s3Region); err != nil {
        eLog.Println("Failed to retrieve S3 object ETag via API")
        eLog.Fatalf("%v", err)
    }
    iLog.Printf("Retrieved ETag information via S3 API")
    iLog.Printf("ETag Value: %s", etag.Value)

    //==================
    // VALIDATE THE FILE
    //==================

    iLog.Printf("Input File: %s", inputFile)
    var calculated *eTag
    var isValid bool
    wLog.Printf("Verifying file integrity. This will take some time.")
    wLog.Printf("Using process count: %v", processCount)
    start := time.Now()
    if calculated, isValid, err = etag.validateFile(inputFile); err != nil {
        eLog.Fatalf("Failed to verify integrity of file: %v", err)
    }
    duration := time.Since(start)

    iLog.Printf("Integrity check finished")
    iLog.Printf("AWS ETag        > %s", etag.Value)
    iLog.Printf("Calculated ETag > %s", calculated.Value)

    var outcome string
    if isValid {
        iLog.Printf("ETag values matched")
        iLog.Printf("File integrity status: confirmed")
        outcome = "verified"
    } else {
        iLog.Print("ETag values mismatched")
        wLog.Printf("File integrity status: compromised")
        outcome = "compromised"
    }

    iLog.Printf("Verification Duration: %v:%v:%v.%v", math.Round(duration.Hours()),
        math.Round(duration.Minutes()), math.Round(duration.Seconds()), duration.Milliseconds())
    fmt.Printf("\nFile integrity: %s\n", outcome)
}

package main

import (
    "crypto/md5"
    "errors"
    "fmt"
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/credentials"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/s3"
    "io"
    "net/url"
    "os"
    "strconv"
    "strings"
)

var (
    hashChan = make(chan *hashValue)
)

// eTag represents an Amazon S3 object ETag value.
type eTag struct {
    // Determines if the file was uploaded multiple chunks, i.e., Multipart.
    Multipart bool
    // Value is the raw ETag value.
    Value string
    // Md5 hash part of the eTag.
    Md5 string
    // ChunkCount is the count of chunks in a Multipart upload.
    ChunkCount int
    ChunkSize  int64
}

// newETag parses a string ETag value to produce an eTag.
func newETag(v string) (*eTag, error) {
    v = strings.Replace(v, "\"", "", -1)
    if md5, chunkCount, err := parseETag(v); err != nil {
        return nil, err
    } else {
        return &eTag{
            Multipart:  chunkCount > 0,
            Value:      v,
            Md5:        md5,
            ChunkCount: chunkCount,
        }, err
    }
}

func getS3ObjectETag(objUrl, region string) (etag *eTag, err error) {

    // Format:
    // https://BUCKET_NAME.s3.amazonaws.com/OBJECT_KEY

    // Parse objUrl to a URL.
    var u *url.URL
    if u, err = url.Parse(objUrl); err != nil {
        return etag, errors.New(fmt.Sprintf("Failed to pare S3 object URL: %v", objUrl))
    }

    bucket := strings.Split(u.Host, ".")[0]
    key := strings.TrimPrefix(u.Path, "/")

    //==============================
    // PULL ETAG INFORMATION FROM S3
    //==============================

    awsConfig := aws.NewConfig()

    if awsConfig.Credentials == nil {
        awsConfig.Credentials = credentials.AnonymousCredentials
    }

    if awsConfig.Region == nil {
        awsConfig.Region = &region
    }

    var sess *session.Session
    if sess, err = session.NewSession(awsConfig); err != nil {
        return etag, errors.New(fmt.Sprintf("Failed to build AWS session to retrieve ETag information: %v", err))
    }

    var obj *s3.HeadObjectOutput
    partNumber := int64(1)
    s3Cli := s3.New(sess)
    if obj, err = s3Cli.HeadObject(&s3.HeadObjectInput{
        Bucket:     &bucket,
        Key:        &key,
        PartNumber: &partNumber,
    }); err != nil {
        return etag, errors.New(fmt.Sprintf("Failed to retrieve ETag information for S3 object: %v", err))
    }

    //==========================
    // PARSE AND RETURN THE ETAG
    //==========================

    if etag, err = newETag(*obj.ETag); err != nil {
        return etag, errors.New(fmt.Sprintf("Failed to parse ETag returned by S3: %v", err))
    }
    etag.ChunkSize = *obj.ContentLength
    return etag, err
}

// parseETag parses an S3 object ETag and returns an error when things go poorly.
// The ETag is formatted as <MD5>-<CHUNK COUNT>, like:
// 1d066c8a194f8f00b833ecc50d699cd9-996
func parseETag(eTag string) (eTagMd5 string, chunks int, err error) {

    //===========================
    // VALIDATE THE MD5 COMPONENT
    //===========================

    split := strings.Split(eTag, "-")
    if !md5Reg.MatchString(split[0]) {
        return eTagMd5, chunks, errors.New("md5 part of ETag failed validation")
    }

    //====================
    // VALIDATE THE CHUNKS
    //====================

    eTagMd5 = split[0]
    switch len(split) {
    case 1:
        // NOP
    case 2:
        var buff int64
        if buff, err = strconv.ParseInt(split[1], 10, 32); err != nil {
            return eTagMd5, chunks, errors.New(fmt.Sprintf("Failed to parse ETag chunk count: %v", err))
        } else {
            chunks = int(buff)
        }
    default:
        err = errors.New(fmt.Sprintf("the ETag component split into too many parts; expected 2, got: %v", len(split)))
    }

    return eTagMd5, chunks, err
}

// fileHandle is used to reference the file being hashed for
// comparison.
type fileHandle struct {
    File *os.File
    Free bool
    Chan chan *md5ChunkArgs
}

// md5ChunkArgs represents a set of arguments that are sent to
// fileHandle processes.
type md5ChunkArgs struct {
    ChunkIndex uint
    Offset     int64
    ChunkSize  int64
}

// hashValue is sent to the final hashing process. ChunkIndex
// is included to ensure that all chunks are hashed in the proper
// sequence.
type hashValue struct {
    ChunkIndex uint
    Value      []byte
}

func newFileHandle(f *os.File) fileHandle {
    c := make(chan *md5ChunkArgs)

    // Server process that receives chunks to hash
    go func() {
        defer f.Close()
        for {
            args := <-c
            if args == nil {
                break
            } else {
                // Seek to file offset
                f.Seek(args.Offset, io.SeekStart)

                // Initialize an MD5 hasher and copy data to it
                buff := md5.New()
                io.CopyN(buff, f, args.ChunkSize)

                // Send the hash value to the final hashing process
                hashChan <- &hashValue{
                    ChunkIndex: args.ChunkIndex,
                    Value:      buff.Sum(nil),
                }
            }
        }
    }()

    return fileHandle{
        File: f,
        Free: true,
        Chan: c,
    }
}

// validateFile hashes the chunks of inputFile to derive a newly
// calculated eTag object for comparison.
func (e eTag) validateFile(inputFile string) (calculated *eTag, isValid bool, err error) {

    //===========================
    // VALIDATE INTEGRITY OF FILE
    //===========================

    chunkCount := 0
    finalMd5 := md5.New()

    if e.ChunkCount == 0 {

        //============================
        // OPEN INPUT FILE FOR READING
        //============================

        var file *os.File
        if file, err = os.Open(inputFile); err != nil {
            return calculated, isValid, errors.New(fmt.Sprintf("Failed to open file for reading: %v", err))
        }
        defer file.Close()

        //==========================
        // HANDLE NON-MULTIPART FILE
        //==========================

        chunkMb := int64(mB)
        for ; err == nil; {
            _, err = io.CopyN(finalMd5, file, chunkMb)
        }
        chunkCount++

    } else {

        //======================
        // HANDLE MULTIPART FILE
        //======================

        // Initialize a file to read for each process
        var fHandles []fileHandle
        for i := 0; i < threadCount; i++ {
            var file *os.File
            if file, err = os.Open(inputFile); err != nil {
                return calculated, isValid, errors.New(fmt.Sprintf("Failed to open file for reading: %v", err))
            }
            fHandles = append(fHandles, newFileHandle(file))
        }

        //===============================================
        // START A GOROUTINE THAT WILL PROCESS ALL HASHES
        //===============================================

        go func() {
            // Track received hash chunks
            chunkHashes := make(map[uint][]byte)
            for {
                // Receive chunk MD5
                v := <-hashChan
                if v == nil {
                    break
                }
                chunkHashes[v.ChunkIndex] = v.Value
            }
            for i := uint(0); i < uint(e.ChunkCount); i++ {
                finalMd5.Write(chunkHashes[i])
            }
            hashChan <- nil
        }()

        //===============================================
        // ROUND ROBIN CHUNK PROCESSING ACROSS GOROUTINES
        //===============================================

        stat, _ := fHandles[0].File.Stat()
        var byteCount int64
        var lastRoutine int
        for ; chunkCount < e.ChunkCount; chunkCount++ {

            if lastRoutine == threadCount {
                lastRoutine = 0
            }

            chunkSize := e.ChunkSize
            if byteCount+e.ChunkSize > stat.Size() {
                chunkSize = stat.Size() - (int64(e.ChunkCount-1) * e.ChunkSize)
            }

            fHandles[lastRoutine].Chan <- &md5ChunkArgs{
                ChunkIndex: uint(chunkCount),
                Offset:     byteCount,
                ChunkSize:  chunkSize,
            }

            // Move processed byte count forward
            byteCount += e.ChunkSize

            lastRoutine++
        }

        //=========
        // CLEAN UP
        //=========

        for _, fh := range fHandles {
            if len(fh.Chan) > 0 {
                <-fh.Chan
            }
            // Tells goroutine that we're shutting down
            fh.Chan <- nil
            close(fh.Chan)
        }

        // Indicate to hashChan that all chunks have been sent
        hashChan <- nil
        <-hashChan
        close(hashChan)
    }

    //===============
    // PREPARE OUTPUT
    //===============

    m := fmt.Sprintf("%x", finalMd5.Sum(nil))
    if e.Multipart {
        calculated, err = newETag(fmt.Sprintf("%s-%v", m, chunkCount))
    } else {
        calculated, err = newETag(m)
    }

    return calculated, m == e.Md5, err
}

// calcETagValue will calculate the ETag value for
func calcETagValue(file *os.File, chunkSize int64) (etag string) {

    stat, err := file.Stat()
    if err != nil {
        eLog.Fatalf("Failed to stat file: %v", err)
    }

    m := md5.New()
    if stat.Size() <= chunkSize {
        if _, err := io.Copy(m, file); err != nil {
            eLog.Fatalf("Failed to read file into hasher: %v", err)
        }
        etag = fmt.Sprintf("%x", m.Sum(nil))
    } else {

        var c, copied int64
        var chunkCount int

        for ; err == nil; {
            buffM := md5.New()
            c, err = io.CopyN(buffM, file, chunkSize)
            if c > 0 {
                m.Write(buffM.Sum(nil))
                copied += c
                chunkCount++
            }
        }

        if err != nil && err != io.EOF {
            eLog.Fatalf("Unhandled exception while reading temp file: %v", err)
        }
        etag = fmt.Sprintf("%x-%v", m.Sum(nil), chunkCount)
    }

    return etag
}

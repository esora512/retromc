package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/leNicDev/retromc/level"
)

const b2FileName = "world.tar.gz"

// b2Client talks to Backblaze B2's native (non-S3) API. Credentials come
// from the KEY_ID, APP_KEY and B2_BUCKET env vars.
type b2Client struct {
	keyID  string
	appKey string
	bucket string
	http   *http.Client
}

func newB2Client() *b2Client {
	keyID := os.Getenv("KEY_ID")
	appKey := os.Getenv("APP_KEY")
	bucket := os.Getenv("B2_BUCKET")
	if keyID == "" || appKey == "" || bucket == "" {
		return nil
	}
	return &b2Client{keyID: keyID, appKey: appKey, bucket: bucket, http: &http.Client{Timeout: 60 * time.Second}}
}

type b2Session struct {
	apiURL      string
	downloadURL string
	authToken   string
	accountID   string
	bucketID    string
}

func (c *b2Client) authorize() (*b2Session, error) {
	req, err := http.NewRequest(http.MethodGet, "https://api.backblazeb2.com/b2api/v2/b2_authorize_account", nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.keyID, c.appKey)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("b2_authorize_account: %s: %s", resp.Status, body)
	}

	var out struct {
		AccountID          string `json:"accountId"`
		AuthorizationToken string `json:"authorizationToken"`
		APIURL             string `json:"apiUrl"`
		DownloadURL        string `json:"downloadUrl"`
		Allowed            struct {
			BucketID string `json:"bucketId"`
		} `json:"allowed"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}

	sess := &b2Session{
		apiURL:      out.APIURL,
		downloadURL: out.DownloadURL,
		authToken:   out.AuthorizationToken,
		accountID:   out.AccountID,
		bucketID:    out.Allowed.BucketID,
	}
	if sess.bucketID == "" {
		bucketID, err := c.resolveBucketID(sess)
		if err != nil {
			return nil, err
		}
		sess.bucketID = bucketID
	}
	return sess, nil
}

// resolveBucketID looks up the bucket ID by name. Only needed if the
// application key isn't already restricted to a single bucket.
func (c *b2Client) resolveBucketID(sess *b2Session) (string, error) {
	reqBody, _ := json.Marshal(map[string]string{
		"accountId":  sess.accountID,
		"bucketName": c.bucket,
	})
	req, err := http.NewRequest(http.MethodPost, sess.apiURL+"/b2api/v2/b2_list_buckets", bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", sess.authToken)
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("b2_list_buckets: %s: %s", resp.Status, body)
	}

	var out struct {
		Buckets []struct {
			BucketID   string `json:"bucketId"`
			BucketName string `json:"bucketName"`
		} `json:"buckets"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", err
	}
	for _, b := range out.Buckets {
		if b.BucketName == c.bucket {
			return b.BucketID, nil
		}
	}
	return "", fmt.Errorf("bucket %q not found", c.bucket)
}

var errB2NotFound = errors.New("b2: file not found")

func (c *b2Client) download(fileName string) ([]byte, error) {
	sess, err := c.authorize()
	if err != nil {
		return nil, err
	}

	u := sess.downloadURL + "/file/" + c.bucket + "/" + url.PathEscape(fileName)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", sess.authToken)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errB2NotFound
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("b2 download: %s: %s", resp.Status, body)
	}
	return body, nil
}

func (c *b2Client) upload(fileName string, data []byte) error {
	sess, err := c.authorize()
	if err != nil {
		return err
	}

	reqBody, _ := json.Marshal(map[string]string{"bucketId": sess.bucketID})
	req, err := http.NewRequest(http.MethodPost, sess.apiURL+"/b2api/v2/b2_get_upload_url", bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", sess.authToken)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("b2_get_upload_url: %s: %s", resp.Status, body)
	}

	var uploadInfo struct {
		UploadURL          string `json:"uploadUrl"`
		AuthorizationToken string `json:"authorizationToken"`
	}
	if err := json.Unmarshal(body, &uploadInfo); err != nil {
		return err
	}

	sum := sha1.Sum(data)
	uploadReq, err := http.NewRequest(http.MethodPost, uploadInfo.UploadURL, bytes.NewReader(data))
	if err != nil {
		return err
	}
	uploadReq.Header.Set("Authorization", uploadInfo.AuthorizationToken)
	uploadReq.Header.Set("X-Bz-File-Name", url.PathEscape(fileName))
	uploadReq.Header.Set("Content-Type", "b2/x-auto")
	uploadReq.Header.Set("X-Bz-Content-Sha1", hex.EncodeToString(sum[:]))
	uploadReq.ContentLength = int64(len(data))

	uploadResp, err := c.http.Do(uploadReq)
	if err != nil {
		return err
	}
	defer uploadResp.Body.Close()
	if uploadResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(uploadResp.Body)
		return fmt.Errorf("b2 upload: %s: %s", uploadResp.Status, respBody)
	}
	return nil
}

// archiveDir tars+gzips dir's contents (relative paths, no leading dir
// component) into a single in-memory blob.
func archiveDir(dir string) ([]byte, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(relPath)
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
	if err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// extractArchive unpacks a tar.gz blob (as produced by archiveDir) into
// destDir, creating it if needed.
func extractArchive(data []byte, destDir string) error {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer gr.Close()
	tr := tar.NewReader(gr)

	cleanDest := filepath.Clean(destDir)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		target := filepath.Join(cleanDest, filepath.FromSlash(hdr.Name))
		if target != cleanDest && !strings.HasPrefix(target, cleanDest+string(os.PathSeparator)) {
			return fmt.Errorf("archive entry %q escapes destination", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}
}

// restoreWorldFromB2 downloads and extracts the last backup into worldDir,
// if one exists. Meant to run once at startup, before the world starts
// accepting connections.
func restoreWorldFromB2(b2 *b2Client, worldDir string) {
	data, err := b2.download(b2FileName)
	if err != nil {
		if errors.Is(err, errB2NotFound) {
			log.Println("B2: no existing backup found, starting with a fresh world")
			return
		}
		log.Println("B2: failed to download backup, starting with a fresh world:", err)
		return
	}
	if err := extractArchive(data, worldDir); err != nil {
		log.Println("B2: failed to extract backup, starting with a fresh world:", err)
		return
	}
	log.Println("B2: restored world from backup")
}

// flushWorld saves all loaded chunks, their entities and online players on the game loop.
func flushWorld(world *level.World) error {
	done := make(chan error, 1)
	world.Enqueue(func() {
		if err := level.SaveMcRegionSync(world, world.WorldDir); err != nil {
			done <- err
			return
		}
		done <- world.SaveAllPlayersSync()
	})
	return <-done
}

// startShutdownSave flushes the world to local disk on SIGTERM/SIGINT (VM / local runs without B2).
func startShutdownSave(world *level.World) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		log.Println("Shutting down, saving world...")
		if err := flushWorld(world); err != nil {
			log.Println("Failed to save world on shutdown:", err)
			os.Exit(1)
		}
		log.Println("World saved")
		os.Exit(0)
	}()
}

// backupWorldToB2 flushes the world to disk and uploads a fresh archive.
func backupWorldToB2(b2 *b2Client, world *level.World) {
	if err := flushWorld(world); err != nil {
		log.Println("B2: failed to flush world before backup:", err)
		return
	}

	data, err := archiveDir(world.WorldDir)
	if err != nil {
		log.Println("B2: failed to archive world:", err)
		return
	}
	if err := b2.upload(b2FileName, data); err != nil {
		log.Println("B2: failed to upload backup:", err)
		return
	}
	log.Printf("B2: backup uploaded (%d bytes)", len(data))
}

// startBackupLoop periodically backs up the world, and does one last backup
// on SIGTERM/SIGINT so a Render redeploy or restart doesn't lose anything.
func startBackupLoop(b2 *b2Client, world *level.World) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		log.Println("Shutting down, backing up world to B2 first...")
		backupWorldToB2(b2, world)
		os.Exit(0)
	}()

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			backupWorldToB2(b2, world)
		}
	}()
}

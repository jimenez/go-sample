package main

import (
	"bytes"
	"database/sql"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestPostObject(t *testing.T) {
	f, err := ioutil.TempFile("", "testdb")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	db, err := sql.Open(dbdriver, f.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = createTable(db)
	if err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(postObject(db))
	defer ts.Close()

	jsonStr := `{"value": "foobar"}`

	res, err := http.Post(ts.URL, "application/json", bytes.NewBufferString(jsonStr))
	if err != nil {
		t.Fatal(err)
	}
	msg, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	// TODO instatiate object and json encode into expected
	expected := "{\"key\":1,\"value\":\"foobar\"}\n"
	if expected != string(msg) {
		t.Errorf("got %q, want %q", msg, expected)
	}

	//TODO test long and empty payload
}

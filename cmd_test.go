package main_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	lfstransfer "github.com/charmbracelet/git-lfs-transfer"
	"github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/assert"
)

func newTestRepo(tb testing.TB) (*git.Repository, string) {
	tb.Helper()
	path := filepath.Join(tb.TempDir(), "repo.git")
	r, err := git.PlainInit(path, true)
	if err != nil {
		tb.Fatal(err)
	}
	return r, path
}

func replaceUserId(s string) string {
	usr, err := user.Current()
	if err != nil {
		return s
	}

	username := usr.Username
	switch runtime.GOOS {
	case "windows":
		username = "unknown"
	}
	s = strings.ReplaceAll(s, "0018ownername=test user\n", fmt.Sprintf("%04xownername=%s\n",
		4+len("ownername=")+len(username+"\n"),
		username))

	s = strings.ReplaceAll(s, "0059ownername d76670443f4d5ecdeea34c12793917498e18e858c6f74cd38c4b794273bb5e28 test user\n",
		fmt.Sprintf("%04xownername d76670443f4d5ecdeea34c12793917498e18e858c6f74cd38c4b794273bb5e28 %s\n",
			4+len("ownername d76670443f4d5ecdeea34c12793917498e18e858c6f74cd38c4b794273bb5e28 ")+len(username+"\n"),
			username))

	return s
}

func TestFailedVerify(t *testing.T) {
	_, path := newTestRepo(t)
	msg := strings.Join(
		[]string{
			"000eversion 1",
			"0000000abatch",
			"0011transfer=ssh",
			"001crefname=refs/heads/main",
			"000100476ca13d52ca70c883e0f0bb101e425a89e8624de51db2d2392593af6a84118090 6",
			"0048ce08b837fe0c499d48935175ddce784e8c372d3cfb1c574fe1caff605d4f0626 32",
			"00000050put-object 6ca13d52ca70c883e0f0bb101e425a89e8624de51db2d2392593af6a84118090",
			"000bsize=6",
			"0001000aabc12300000050put-object ce08b837fe0c499d48935175ddce784e8c372d3cfb1c574fe1caff605d4f0626",
			"000csize=32",
			"00010024This is\x00a complicated\xc2\xa9message.",
			"00000053verify-object 6ca13d52ca70c883e0f0bb101e425a89e8624de51db2d2392593af6a84118090",
			"000bsize=5",
			"0000",
		}, "\n",
	)
	expected := strings.Join(
		[]string{
			"000eversion=1",
			"000clocking",
			"0000000fstatus 200",
			"00010000000fstatus 200",
			"0001004e6ca13d52ca70c883e0f0bb101e425a89e8624de51db2d2392593af6a84118090 6 upload",
			"004fce08b837fe0c499d48935175ddce784e8c372d3cfb1c574fe1caff605d4f0626 32 upload",
			"0000000fstatus 200",
			"0000000fstatus 200",
			"0000000fstatus 409",
			"00010012size mismatch",
			"0000",
		}, "\n",
	)

	var out bytes.Buffer
	in := strings.NewReader(msg)
	if err := lfstransfer.Run(in, &out, path, "upload"); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, expected, out.String())
}

func TestMissingObject(t *testing.T) {
	_, path := newTestRepo(t)
	msg := strings.Join(
		[]string{
			"000eversion 1",
			"0000000abatch",
			"0011transfer=ssh",
			"001crefname=refs/heads/main",
			"000100476ca13d52ca70c883e0f0bb101e425a89e8624de51db2d2392593af6a84118090 6",
			"0048ce08b837fe0c499d48935175ddce784e8c372d3cfb1c574fe1caff605d4f0626 32",
			"00000050put-object 6ca13d52ca70c883e0f0bb101e425a89e8624de51db2d2392593af6a84118090",
			"000bsize=6",
			"0001000aabc12300000050put-object ce08b837fe0c499d48935175ddce784e8c372d3cfb1c574fe1caff605d4f0626",
			"000csize=32",
			"00010024This is\x00a complicated\xc2\xa9message.",
			"00000053verify-object 0000000000000000000000000000000000000000000000000000000000000000",
			"000bsize=5",
			"0000",
		}, "\n",
	)
	expected := strings.Join(
		[]string{
			"000eversion=1",
			"000clocking",
			"0000000fstatus 200",
			"00010000000fstatus 200",
			"0001004e6ca13d52ca70c883e0f0bb101e425a89e8624de51db2d2392593af6a84118090 6 upload",
			"004fce08b837fe0c499d48935175ddce784e8c372d3cfb1c574fe1caff605d4f0626 32 upload",
			"0000000fstatus 200",
			"0000000fstatus 200",
			"0000000fstatus 404",
			"0001000enot found",
			"0000",
		}, "\n",
	)

	var out bytes.Buffer
	in := strings.NewReader(msg)
	if err := lfstransfer.Run(in, &out, path, "upload"); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, expected, out.String())
}

func TestSimpleUpload(t *testing.T) {
	_, path := newTestRepo(t)
	msg := strings.Join(
		[]string{
			"000eversion 1",
			"0000000abatch",
			"0011transfer=ssh",
			"0015hash-algo=sha256",
			"001crefname=refs/heads/main",
			"000100476ca13d52ca70c883e0f0bb101e425a89e8624de51db2d2392593af6a84118090 6",
			"0048ce08b837fe0c499d48935175ddce784e8c372d3cfb1c574fe1caff605d4f0626 32",
			"00000050put-object 6ca13d52ca70c883e0f0bb101e425a89e8624de51db2d2392593af6a84118090",
			"000bsize=6",
			"0001000aabc12300000050put-object ce08b837fe0c499d48935175ddce784e8c372d3cfb1c574fe1caff605d4f0626",
			"000csize=32",
			"00010024This is\x00a complicated\xc2\xa9message.",
			"00000053verify-object 6ca13d52ca70c883e0f0bb101e425a89e8624de51db2d2392593af6a84118090",
			"000bsize=6",
			"00000053verify-object ce08b837fe0c499d48935175ddce784e8c372d3cfb1c574fe1caff605d4f0626",
			"000csize=32",
			"0000",
		}, "\n",
	)
	expected := strings.Join(
		[]string{
			"000eversion=1",
			"000clocking",
			"0000000fstatus 200",
			"00010000000fstatus 200",
			"0001004e6ca13d52ca70c883e0f0bb101e425a89e8624de51db2d2392593af6a84118090 6 upload",
			"004fce08b837fe0c499d48935175ddce784e8c372d3cfb1c574fe1caff605d4f0626 32 upload",
			"0000000fstatus 200",
			"0000000fstatus 200",
			"0000000fstatus 200",
			"0000000fstatus 200",
			"0000",
		}, "\n",
	)

	var out bytes.Buffer
	in := strings.NewReader(msg)
	if err := lfstransfer.Run(in, &out, path, "upload"); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, expected, out.String())
}

func TestInvalidHashAlgo(t *testing.T) {
	_, path := newTestRepo(t)
	msg := strings.Join(
		[]string{
			"000eversion 1",
			"0000000abatch",
			"0011transfer=ssh",
			"0015hash-algo=sha512",
			"001crefname=refs/heads/main",
			"000100476ca13d52ca70c883e0f0bb101e425a89e8624de51db2d2392593af6a84118090 6",
			"0048ce08b837fe0c499d48935175ddce784e8c372d3cfb1c574fe1caff605d4f0626 32",
			"0000",
		}, "\n",
	)
	expected := strings.Join(
		[]string{
			"000eversion=1",
			"000clocking",
			"0000000fstatus 200",
			"00010000000fstatus 405",
			"0001003berror: not allowed: unsupported hash algorithm: sha512",
			"0000",
		}, "\n",
	)

	var out bytes.Buffer
	in := strings.NewReader(msg)
	if err := lfstransfer.Run(in, &out, path, "upload"); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, expected, out.String())
}

func TestSimpleDownload(t *testing.T) {
	_, path := newTestRepo(t)
	msg := strings.Join(
		[]string{
			"000eversion 1",
			"0000000abatch",
			"0011transfer=ssh",
			"001crefname=refs/heads/main",
			"000100476ca13d52ca70c883e0f0bb101e425a89e8624de51db2d2392593af6a84118090 6",
			"0048ce08b837fe0c499d48935175ddce784e8c372d3cfb1c574fe1caff605d4f0626 32",
			"00000050put-object ce08b837fe0c499d48935175ddce784e8c372d3cfb1c574fe1caff605d4f0626",
			"000csize=32",
			"00010024This is\x00a complicated\xc2\xa9message.",
			"00000053verify-object ce08b837fe0c499d48935175ddce784e8c372d3cfb1c574fe1caff605d4f0626",
			"000csize=32",
			"0000",
		}, "\n",
	)
	expected := strings.Join(
		[]string{
			"000eversion=1",
			"000clocking",
			"0000000fstatus 200",
			"00010000000fstatus 200",
			"0001004e6ca13d52ca70c883e0f0bb101e425a89e8624de51db2d2392593af6a84118090 6 upload",
			"004fce08b837fe0c499d48935175ddce784e8c372d3cfb1c574fe1caff605d4f0626 32 upload",
			"0000000fstatus 200",
			"0000000fstatus 200",
			"0000",
		}, "\n",
	)

	var out bytes.Buffer
	in := strings.NewReader(msg)
	if err := lfstransfer.Run(in, &out, path, "upload"); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, expected, out.String())

	msg = strings.Join(
		[]string{
			"000eversion 1",
			"0000000abatch",
			"0011transfer=ssh",
			"001crefname=refs/heads/main",
			"000100476ca13d52ca70c883e0f0bb101e425a89e8624de51db2d2392593af6a84118090 6",
			"0048ce08b837fe0c499d48935175ddce784e8c372d3cfb1c574fe1caff605d4f0626 32",
			"00000050get-object ce08b837fe0c499d48935175ddce784e8c372d3cfb1c574fe1caff605d4f0626",
			"0000",
		}, "\n",
	)
	expected = strings.Join(
		[]string{
			"000eversion=1",
			"000clocking",
			"0000000fstatus 200",
			"00010000000fstatus 200",
			"0001004c6ca13d52ca70c883e0f0bb101e425a89e8624de51db2d2392593af6a84118090 6 noop",
			"0051ce08b837fe0c499d48935175ddce784e8c372d3cfb1c574fe1caff605d4f0626 32 download",
			"0000000fstatus 200",
			"000csize=32",
			"00010024This is\x00a complicated\xc2\xa9message.",
			"0000",
		}, "\n",
	)

	out.Reset()
	in = strings.NewReader(msg)
	if err := lfstransfer.Run(in, &out, path, "download"); err != nil {
		t.Fatal(err)
	}

	t.Log(path)
	ls, _ := exec.Command("ls", "-la", path+"/lfs/objects/ce/08").Output()
	t.Log(string(ls))

	bts, err := os.ReadFile(filepath.Join(path, "lfs", "objects", "ce", "08", "ce08b837fe0c499d48935175ddce784e8c372d3cfb1c574fe1caff605d4f0626"))
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "This is\x00a complicated\xc2\xa9message.\n", string(bts))
	assert.Equal(t, expected, out.String())
}

func TestInvalidUpload(t *testing.T) {
	_, path := newTestRepo(t)
	msg := strings.Join(
		[]string{
			"000eversion 1",
			"0000000abatch",
			"0011transfer=ssh",
			"001crefname=refs/heads/main",
			"000100476ca13d52ca70c883e0f0bb101e425a89e8624de51db2d2392593af6a84118090 6",
			"0048ce08b837fe0c499d48935175ddce784e8c372d3cfb1c574fe1caff605d4f0626 32",
			"00000050put-object 6ca13d52ca70c883e0f0bb101e425a89e8624de51db2d2392593af6a84118090",
			"000bsize=6",
			"0001000aabc12300000050put-object ce08b837fe0c499d48935175ddce784e8c372d3cfb1c574fe1caff605d4f0626",
			"000csize=32",
			"00010024This is\x01a complicated\xc2\xa9message.",
			"00000053verify-object 6ca13d52ca70c883e0f0bb101e425a89e8624de51db2d2392593af6a84118090",
			"000bsize=6",
			"00000053verify-object ce08b837fe0c499d48935175ddce784e8c372d3cfb1c574fe1caff605d4f0626",
			"000csize=32",
			"0000",
		}, "\n",
	)
	expected := strings.Join(
		[]string{
			"000eversion=1",
			"000clocking",
			"0000000fstatus 200",
			"00010000000fstatus 200",
			"0001004e6ca13d52ca70c883e0f0bb101e425a89e8624de51db2d2392593af6a84118090 6 upload",
			"004fce08b837fe0c499d48935175ddce784e8c372d3cfb1c574fe1caff605d4f0626 32 upload",
			"0000000fstatus 200",
			"0000000fstatus 400",
			"000100bcerror: corrupt data: invalid object ID, expected ce08b837fe0c499d48935175ddce784e8c372d3cfb1c574fe1caff605d4f0626, got 367988c7cb91e13beda0a15fb271afcbf02fa7a0e75d9e25ac50b2b4b38af5f5",
			"0000000fstatus 200",
			"0000000fstatus 404",
			"0001000enot found",
			"0000",
		}, "\n",
	)

	var out bytes.Buffer
	in := strings.NewReader(msg)
	if err := lfstransfer.Run(in, &out, path, "upload"); err != nil {
		t.Fatal(err)
	}

	bts, err := os.ReadFile(filepath.Join(path, "lfs", "objects", "6c", "a1", "6ca13d52ca70c883e0f0bb101e425a89e8624de51db2d2392593af6a84118090"))
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "abc123", string(bts))
	fp := filepath.Join(path, "lfs", "objects", "ce", "08", "ce08b837fe0c499d48935175ddce784e8c372d3cfb1c574fe1caff605d4f0626")
	if _, err := os.Stat(fp); err == nil {
		t.Errorf("file should not exist %s", fp)
	}

	fp = filepath.Join(path, "lfs", "objects", "36", "79", "367988c7cb91e13beda0a15fb271afcbf02fa7a0e75d9e25ac50b2b4b38af5f5")
	if _, err := os.Stat(fp); err == nil {
		t.Errorf("file should not exist %s", fp)
	}

	assert.Equal(t, expected, out.String())
}

func TestSimpleLocking(t *testing.T) {
	_, path := newTestRepo(t)
	now := time.Now().UTC()
	msg := strings.Join(
		[]string{
			"000eversion 1",
			"00000009lock",
			"000dpath=foo",
			"001crefname=refs/heads/main",
			"00000009lock",
			"000dpath=foo",
			"001crefname=refs/heads/main",
			"0000000elist-lock",
			"000elimit=100",
			"0000004cunlock d76670443f4d5ecdeea34c12793917498e18e858c6f74cd38c4b794273bb5e28",
			"0000",
		}, "\n",
	)
	expected := strings.Join(
		[]string{
			"000eversion=1",
			"000clocking",
			"0000000fstatus 200",
			"00010000000fstatus 201",
			"0048id=d76670443f4d5ecdeea34c12793917498e18e858c6f74cd38c4b794273bb5e28",
			"000dpath=foo",
			"0023locked-at=" + now.Format(time.RFC3339),
			"0018ownername=test user",
			"0000000fstatus 409",
			"0048id=d76670443f4d5ecdeea34c12793917498e18e858c6f74cd38c4b794273bb5e28",
			"000dpath=foo",
			"0023locked-at=" + now.Format(time.RFC3339),
			"0018ownername=test user",
			"0001000dconflict",
			"0000000fstatus 200",
			"0001004alock d76670443f4d5ecdeea34c12793917498e18e858c6f74cd38c4b794273bb5e28",
			"004epath d76670443f4d5ecdeea34c12793917498e18e858c6f74cd38c4b794273bb5e28 foo",
			"0064locked-at d76670443f4d5ecdeea34c12793917498e18e858c6f74cd38c4b794273bb5e28 " + now.Format(time.RFC3339),
			"0059ownername d76670443f4d5ecdeea34c12793917498e18e858c6f74cd38c4b794273bb5e28 test user",
			"0050owner d76670443f4d5ecdeea34c12793917498e18e858c6f74cd38c4b794273bb5e28 ours",
			"0000000fstatus 200",
			"0048id=d76670443f4d5ecdeea34c12793917498e18e858c6f74cd38c4b794273bb5e28",
			"000dpath=foo",
			"0023locked-at=" + now.Format(time.RFC3339),
			"0018ownername=test user",
			"0000",
		}, "\n",
	)

	var out bytes.Buffer
	in := strings.NewReader(msg)
	if err := lfstransfer.Run(in, &out, path, "upload"); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, replaceUserId(expected), out.String())
}

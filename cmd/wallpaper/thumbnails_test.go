package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// startedJobs records what the state command would start; no test starts a real process.
var startedJobs [][]string

func init() {
	startDetached = func(args []string) error {
		startedJobs = append(startedJobs, args)
		return nil
	}
}

func takeStartedJobs() [][]string {
	jobs := startedJobs
	startedJobs = nil
	return jobs
}

func removeThumbs(t *testing.T, g gallery) {
	t.Helper()
	for _, name := range []string{"0001.png", "0002.png", "0003.png"} {
		if err := os.Remove(filepath.Join(g.appDir, "data", "thumbs", name)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestStateStartsTheThumbnailJobInsteadOfMakingThumbnails(t *testing.T) {
	g := importedGallery(t)
	removeThumbs(t, g)
	takeStartedJobs()
	reply := g.state(t)
	for _, picture := range reply.Pictures {
		if picture.Thumb != "" {
			t.Errorf("picture %s: the state read must not make a thumbnail", picture.ID)
		}
	}
	jobs := takeStartedJobs()
	if len(jobs) != 1 || jobs[0][0] != "thumbnails" || !slices.Contains(jobs[0], g.appDir) ||
		!slices.Contains(jobs[0], filepath.Join(g.appDir, "thumbs.pid")) {
		t.Fatalf("started %v, want one thumbnails job for this app", jobs)
	}
}

func TestStateStartsNoThumbnailJobWhenNothingIsMissing(t *testing.T) {
	g := importedGallery(t)
	takeStartedJobs()
	g.state(t)
	if jobs := takeStartedJobs(); len(jobs) != 0 {
		t.Fatalf("started %v with every thumbnail present", jobs)
	}
}

func TestStateStartsNoThumbnailJobWhileAnotherJobRuns(t *testing.T) {
	g := importedGallery(t)
	removeThumbs(t, g)
	takeStartedJobs()
	write(t, filepath.Join(g.appDir, "thumbs.pid"), []byte(strconvPid()))
	g.state(t)
	write(t, filepath.Join(g.appDir, "none.pid"), []byte(strconvPid())) // an import runs
	os.Remove(filepath.Join(g.appDir, "thumbs.pid"))
	g.state(t)
	if jobs := takeStartedJobs(); len(jobs) != 0 {
		t.Fatalf("started %v while a job was running", jobs)
	}
}

func TestThumbnailsCommandMakesTheMissingThumbnails(t *testing.T) {
	g := importedGallery(t)
	removeThumbs(t, g)
	if status, output := g.run(t, "thumbnails"); status != 0 {
		t.Fatalf("thumbnails: %d %s", status, output)
	}
	for _, picture := range g.state(t).Pictures {
		if picture.Thumb == "" {
			t.Errorf("picture %s still has no thumbnail", picture.ID)
		}
	}
	if _, err := os.Stat(filepath.Join(g.appDir, "thumbs.pid")); !os.IsNotExist(err) {
		t.Errorf("the job must release its lock: %v", err)
	}
}

func TestThumbnailsCommandLeavesTheWorkToARunningJob(t *testing.T) {
	g := importedGallery(t)
	removeThumbs(t, g)
	write(t, filepath.Join(g.appDir, "thumbs.pid"), []byte(strconvPid()))
	if status, output := g.run(t, "thumbnails"); status != 0 {
		t.Fatalf("thumbnails: %d %s", status, output)
	}
	if _, err := os.Stat(filepath.Join(g.appDir, "data", "thumbs", "0001.png")); err == nil {
		t.Error("a second job must not run while the first holds the lock")
	}
}

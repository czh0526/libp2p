package test_goprocess

import (
	"testing"

	"github.com/jbenet/goprocess"
)

type tree struct {
	goprocess.Process
	c []tree
}

func setupHierrarchy(p goprocess.Process) tree {
	t := func(n goprocess.Process, ts ...tree) tree {
		return tree{n, ts}
	}

	a := goprocess.WithParent(p)
	b1 := goprocess.WithParent(a)
	b2 := goprocess.WithParent(a)
	c1 := goprocess.WithParent(b1)
	c2 := goprocess.WithParent(b1)
	c3 := goprocess.WithParent(b2)
	c4 := goprocess.WithParent(b2)

	return t(a, t(b1, t(c1), t(c2)), t(b2, t(c3), t(c4)))
}

func TestClosingClosed(t *testing.T) {

	bWait := make(chan struct{})
	a := goprocess.WithParent(goprocess.Background())
	a.Go(func(proc goprocess.Process) {
		<-bWait
	})

	Q := make(chan string, 3)

	go func() {
		<-a.Closing()
		Q <- "closing"
		bWait <- struct{}{}
	}()

	go func() {
		<-a.Closed()
		Q <- "closed"
	}()

	go func() {
		a.Close()
		Q <- "closed"
	}()

	if q := <-Q; q != "closing" {
		t.Error("order incorrect. closing not first")
	}
	if q := <-Q; q != "closed" {
		t.Error("order incorrect. closed not second.")
	}
	if q := <-Q; q != "closed" {
		t.Error("order incorrect. closed not third.")
	}
}

package main

import (
	"testing"
	"time"
)

// вспомогательная функция для создания done-канала который закрывается через delay
func sig(delay time.Duration) <-chan interface{} {
	c := make(chan interface{})
	go func() {
		defer close(c)
		time.Sleep(delay)
	}()
	return c
}

func TestOr_NoChannels(t *testing.T) {
	if or() != nil {
		t.Errorf("expected nil when no channels passed")
	}
}

func TestOr_OneChannel(t *testing.T) {
	ch := make(chan interface{})
	go func() {
		defer close(ch)
	}()

	if or(ch) != ch {
		t.Errorf("expected or(ch) to return the same channel")
	}
}

func TestOr_TwoChannels(t *testing.T) {
	start := time.Now()

	<-or(
		sig(100*time.Millisecond),
		sig(1*time.Second),
	)

	elapsed := time.Since(start)
	if elapsed > 200*time.Millisecond {
		t.Errorf("or did not trigger fast enough: took %v", elapsed)
	}
}

func TestOr_ManyChannels(t *testing.T) {
	start := time.Now()

	<-or(
		sig(3*time.Second),
		sig(2*time.Second),
		sig(100*time.Millisecond),
		sig(4*time.Second),
		sig(5*time.Second),
	)

	elapsed := time.Since(start)
	if elapsed > 200*time.Millisecond {
		t.Errorf("or should have triggered in ~100ms, took %v", elapsed)
	}
}

func TestOr_AlreadyClosedChannel(t *testing.T) {
	ch := make(chan interface{})
	close(ch)

	start := time.Now()

	<-or(ch, sig(1*time.Hour))

	elapsed := time.Since(start)
	if elapsed > 10*time.Millisecond {
		t.Errorf("with an already closed channel, or should fire immediately, took %v", elapsed)
	}
}

func TestOr_ChannelOrderDoesNotMatter(t *testing.T) {
	start := time.Now()

	<-or(
		sig(2*time.Second),
		sig(3*time.Second),
		sig(50*time.Millisecond),
		sig(4*time.Second),
	)

	elapsed := time.Since(start)
	if elapsed > 100*time.Millisecond {
		t.Errorf("or took too long: %v", elapsed)
	}
}

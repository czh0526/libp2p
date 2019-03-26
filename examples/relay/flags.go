package main

import "flag"

func parseFlags(n *int, raddr *string, target *string) {
	flag.IntVar(n, "n", 0, "this node no, should be 1, 2, 3")
	flag.StringVar(raddr, "raddr", "", "this is relay node address")
	flag.StringVar(target, "target", "", "this is a target node")
	flag.Parse()

	if *n == 0 {
		panic("n should be 1,2,3.")

	} else if *n == 1 {
		if *raddr == "" {
			panic("relay address can't be empty")
		}
		if *target == "" {
			panic("target address can't be empty")
		}
	} else if *n == 3 {
		if *raddr == "" {
			panic("relay address can't be empty")
		}
	}
}

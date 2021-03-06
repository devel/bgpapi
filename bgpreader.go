package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

var neighbors Neighbors

func bgpReader() {

	neighbors = make(Neighbors)

	r := bufio.NewReader(os.Stdin)

	var err error
	for line, err := r.ReadString('\n'); err == nil; line, err = r.ReadString('\n') {
		line = strings.TrimSpace(line)
		if len(line) > 0 {
			// fmt.Println("X", line)
			f := strings.SplitN(line, " ", 4)
			// fmt.Printf("%#v\n", f)

			neighbor_ip := f[1]
			command := f[2]

			if neighbors[neighbor_ip] == nil {
				neighbor := new(Neighbor)
				neighbors[neighbor_ip] = neighbor
				// neighbor.AsnPrefix = new(map[ASN]Prefixes)
				// neighbor.PrefixAsn = make(map[string]ASN)
			}

			neighbor := neighbors[neighbor_ip]

			switch command {
			case "up", "connected":
				neighbor.State = command
			case "update":
				neighbor.State = "update " + f[3]
			case "announced":
				// fmt.Printf("R: %#v\n", r)

				neighbor.Updates++

				route := parseRoute(f[3])

				if ones, _ := route.Prefix.Mask.Size(); ones < 8 || ones > 25 {
					// fmt.Println("prefix mask too big or small", route.Prefix)
				} else {
					if neighbor.AsnPrefix == nil {
						neighbor.AsnPrefix = make(map[ASN]Prefixes)
					}
					if neighbor.PrefixAsn == nil {
						neighbor.PrefixAsn = make(Prefixes)
					}

					if neighbor.AsnPrefix[route.PrimaryASN] == nil {
						neighbor.AsnPrefix[route.PrimaryASN] = make(Prefixes)
					}

					neighbor.AsnPrefix[route.PrimaryASN][route.Prefix.String()] = 0
					neighbor.PrefixAsn[route.Prefix.String()] = route.PrimaryASN
				}
			case "withdrawn":

				neighbor.Updates++

				// fmt.Println("withdraw", f[3])
				route := parseRoute(f[3])

				// x, y := neighbor.PrefixAsn[route.Prefix.String()]
				// fmt.Println("X/Y", x, y)

				if asn, exists := neighbor.PrefixAsn[route.Prefix.String()]; exists {
					// fmt.Println("Removing ASN from prefix", asn, route.Prefix)
					delete(neighbor.PrefixAsn, route.Prefix.String())
					delete(neighbor.AsnPrefix[asn], route.Prefix.String())
				} else {
					fmt.Println("Could not find prefix in PrefixAsn")
					fmt.Println("%#v", neighbor.PrefixAsn)
				}

			default:
				fmt.Println("Unknown command:", command)
				fmt.Println("LINE:", line)
				panic("Command not implemented:")
			}
		}
	}

	if err != nil && err != io.EOF {
		fmt.Println(err)
		return
	} else {
		fmt.Println("EOF")
	}
}

func parseRoute(input string) *Route {

	r := strings.Split(input, " ")

	route := new(Route)
	route.Options = make(map[string]string)
	aspath := make(ASPath, 0)

	var key string

	state := parseKey

	for _, v := range r {
		// fmt.Printf("k: %s, v: %s, state: %#v\n", key, v, state)

		switch state {
		case parseKey:
			{
				state = parseValue
				key = v
				continue
			}
		case parseValue:
			if v == "[" {
				state = parseList
				continue
			}
			state = parseKey

			if key == "as-path" {
				addASPath(&aspath, v)
			}
			route.Options[key] = v
			continue
		case parseList:
			{
				if v == "]" {
					state = parseKey
					continue
				}
				if key != "as-path" {
					fmt.Printf("key: %s, v: %s\n\n", key, v)
					panic("can only do list for as-path")
				}
				if v == "(" {
					state = parseSkip
					continue
				}

				addASPath(&aspath, v)

			}
		case parseSkip:
			if v == ")" {
				state = parseList
			}
		}
	}
	// fmt.Printf("%#v / %#v\n", route, aspath)

	_, prefix, err := net.ParseCIDR(route.Options["route"])
	if err != nil {
		fmt.Printf("Could not parse prefix %s %e\n", route.Options["route"], err)
		panic("bad prefix")
	}
	route.Prefix = prefix
	// fmt.Printf("IP: %s, PREFIX: %s\n", ip, prefix)

	if len(aspath) > 0 {
		route.PrimaryASN = ASN(aspath[len(aspath)-1])
	}

	if DEBUG {
		fmt.Println("PREFIX", route.Prefix)
	}

	return route
}

func addASPath(aspath *ASPath, v string) {
	asn, err := strconv.Atoi(v)
	if err != nil {
		fmt.Println("Could not parse number", v)
		panic("Bad as-path")
	}
	*aspath = append(*aspath, ASN(asn))
}

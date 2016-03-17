package quorum

import (
	"sort"
	"strconv"
	"strings"
)

func (this HostPort) toServer() (*Server, error) {
	s := strings.Split(string(this), ":")
	server := &Server{Ip: s[0]}

	if len(s) > 1 {
		p, err := strconv.Atoi(s[1])
		if err != nil {
			return nil, err
		} else {
			server.Port = p
		}
	}
	return server, nil
}

type serverSorter struct {
	servers []*Server
}

func (s *serverSorter) Sort() {
	sort.Sort(s)
}

func (s *serverSorter) Add(sv *Server) {
	if s.servers == nil {
		s.servers = []*Server{}
	}
	s.servers = append(s.servers, sv)
}

// Len is part of sort.Interface.
func (s *serverSorter) Len() int {
	return len(s.servers)
}

// Swap is part of sort.Interface.
func (s *serverSorter) Swap(i, j int) {
	s.servers[i], s.servers[j] = s.servers[j], s.servers[i]
}

// Less is part of sort.Interface. It is implemented by calling the "by" closure in the sorter.
func (s *serverSorter) Less(i, j int) bool {
	if s.servers[i].Ip == s.servers[j].Ip {
		return s.servers[i].Port < s.servers[j].Port
	} else {
		return s.servers[i].Ip < s.servers[j].Ip
	}
}

func dedupAndSort(a []HostPort, b ...[]HostPort) ([]*Server, error) {
	seen := map[string]interface{}{}
	out := []*Server{}
	for _, h := range a {
		seen[string(h)] = 1
		s, err := h.toServer()
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	for _, l := range b {
		for _, h := range l {
			if _, exists := seen[string(h)]; !exists {
				seen[string(h)] = 1
				s, err := h.toServer()
				if err != nil {
					return nil, err
				}
				out = append(out, s)
			}
		}
	}
	sorter := &serverSorter{servers: out}
	sort.Sort(sorter)
	return sorter.servers, nil
}

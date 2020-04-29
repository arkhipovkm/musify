package vk

import (
	"strings"
	"sync"
)

func SectionQuery(query string, offset, n int, u *User) (map[string]*Playlist, []*Playlist, []*Audio, error) {
	var err error
	playlistIDs, lastPlaylist, err := alSection(query, u)
	if err != nil {
		return nil, nil, nil, err
	}
	if lastPlaylist == nil {
		return nil, nil, nil, err
	}
	if len(lastPlaylist.List) < offset {
		lastPlaylist.List = nil
	} else {
		lastPlaylist.List = lastPlaylist.List[offset:]
	}

	if len(lastPlaylist.List) > n {
		lastPlaylist.List = lastPlaylist.List[:n]
	}

	uniquePlaylistIDs := make(map[string]bool)
	for _, a := range lastPlaylist.List {
		if a.Album != "" {
			uniquePlaylistIDs[a.Album] = true
		}
	}
	if offset == 0 {
		for _, plID := range playlistIDs {
			uniquePlaylistIDs[plID] = true
		}
	}

	ch := make(chan *Playlist, len(uniquePlaylistIDs))
	wg := &sync.WaitGroup{}
	for plID := range uniquePlaylistIDs {
		go LoadPlaylistChan(plID, u, ch)
	}

	wg.Add(1)
	go lastPlaylist.AcquireURLsWG(u, wg)

	playlistMap := make(map[string]*Playlist)
	m := len(uniquePlaylistIDs)
	for i := 0; i < m; i++ {
		pl := <-ch
		playlistMap[pl.FullID()] = pl
	}
	wg.Wait()
	lastPlaylist.DecypherURLs(u)

	var topPlaylists []*Playlist
	if offset == 0 {
		for _, plID := range playlistIDs {
			if len(strings.Split(plID, "_")) == 2 {
				plID = plID + "_"
			}
			topPlaylists = append(topPlaylists, playlistMap[plID])
		}
	}

	return playlistMap, topPlaylists, lastPlaylist.List, err
}

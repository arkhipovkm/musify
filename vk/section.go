package vk

import (
	"sync"
)

func SectionQuery(query string, offset, n int, u *User) (map[string]Playlist, []*Audio, error) {
	var err error
	playlistIDs, lastPlaylist, err := alSection(query, u)
	if err != nil {
		return nil, nil, err
	}
	lastPlaylist.List = lastPlaylist.List[offset : offset+n]

	uniquePlaylistIDs := make(map[string]bool)
	for _, a := range lastPlaylist.List {
		uniquePlaylistIDs[a.Album] = true
	}
	for _, plID := range playlistIDs {
		uniquePlaylistIDs[plID] = true
	}

	ch := make(chan *Playlist, len(uniquePlaylistIDs))
	wg := &sync.WaitGroup{}
	for plID := range uniquePlaylistIDs {
		go LoadPlaylistChan(plID, u, ch)
	}

	wg.Add(1)
	go lastPlaylist.AcquireURLsWG(u, wg)

	playlistMap := make(map[string]Playlist)
	m := len(uniquePlaylistIDs)
	for i := 0; i < m; i++ {
		pl := <-ch
		playlistMap[pl.FullID()] = *pl
	}
	wg.Wait()
	lastPlaylist.DecypherURLs(u)
	return playlistMap, lastPlaylist.List, err
}

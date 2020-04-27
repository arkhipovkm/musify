package vk

import (
	"fmt"
	"sync"
)

func SectionQuery(query string, offset, n int, u *User) (map[string]Playlist, []Audio) {
	playlistIDs, lastPlaylist := alSection(query, u)

	lastPlaylist.List = lastPlaylist.List[offset : offset+n]

	uniquePlaylistIDs := make(map[string]bool)
	for _, a := range lastPlaylist.List {
		uniquePlaylistIDs[a.Album] = true
	}
	for _, plID := range playlistIDs {
		uniquePlaylistIDs[plID] = true
	}
	fmt.Printf("About to fetch %d unique playlists present in the section..\n", len(uniquePlaylistIDs))

	ch := make(chan Playlist, len(uniquePlaylistIDs))
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
		playlistMap[pl.FullID()] = pl
	}
	wg.Wait()
	fmt.Println("Fetched playlists: ", len(playlistMap))
	lastPlaylist.DecypherURLs(u)
	return playlistMap, lastPlaylist.List
}

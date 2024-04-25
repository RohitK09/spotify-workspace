// This example demonstrates how to authenticate with Spotify using the authorization code flow.
// In order to run this example yourself, you'll need to:
//
//  1. Register an application at: https://developer.spotify.com/my-applications/
//     - Use "http://localhost:8080/callback" as the redirect URI
//  2. Set the SPOTIFY_ID environment variable to the client ID you got in step 1.
//  3. Set the SPOTIFY_SECRET environment variable to the client secret from step 1.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/samber/lo"
	spotifyauth "github.com/zmb3/spotify/v2/auth"

	"github.com/zmb3/spotify/v2"
)

// redirectURI is the OAuth redirect URI for the application.
// You must register an application at Spotify's developer portal
// and enter this value.
const redirectURI = "<redirect-uri>"

var (
	auth = spotifyauth.New(spotifyauth.WithRedirectURL(redirectURI), spotifyauth.WithScopes(spotifyauth.ScopeUserLibraryRead, spotifyauth.ScopePlaylistModifyPrivate, spotifyauth.ScopePlaylistReadPrivate),
		spotifyauth.WithClientID("client_id"), spotifyauth.WithClientSecret("client_secret"))
	ch       = make(chan *spotify.Client)
	state    = "abc123"
	maxLimit = 50
)

func main() {
	// first start an HTTP server
	http.HandleFunc("/callback", completeAuth)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got request for:", r.URL.String())
	})
	go func() {
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			log.Fatal(err)
		}
	}()

	url := auth.AuthURL(state)
	fmt.Println("Please log in to Spotify by visiting the following page in your browser:", url)

	// wait for auth to complete
	client := <-ch

	// use the client to make calls that require authorization
	user, err := client.CurrentUser(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("You are logged in as:", user.ID)
}

func completeAuth(w http.ResponseWriter, r *http.Request) {
	tok, err := auth.Token(r.Context(), state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, state)
	}

	// use the token to get an authenticated client
	client := spotify.New(auth.Client(r.Context(), tok))
	fmt.Fprintf(w, "Login Completed!")

	res, err := client.CurrentUsersTracks(context.Background(), spotify.Limit(maxLimit))
	if err != nil {
		fmt.Print(err)
	}
	prevOffset := res.Offset
	//keep getting on data till there are more pages of data
	songDict := make(map[int][]spotify.ID)

	for res.Next != "" {
		res, _ = client.CurrentUsersTracks(context.Background(), spotify.Limit(maxLimit), spotify.Offset(prevOffset))
		prevOffset = res.Offset + len(res.Tracks)
		songDict = lo.Assign(appendDictBasedOnYear(res.Tracks), songDict)
	}
	for k, v := range songDict {
		createPlayListBasedOnYear(client, v, k)
	}
	ch <- client

}

func appendDictBasedOnYear(tracks []spotify.SavedTrack) map[int][]spotify.ID {
	tracksByYear := make(map[int][]spotify.ID)
	for _, track := range tracks {
		addedTime, _ := time.Parse(spotify.TimestampLayout, track.AddedAt)
		tracksByYear[addedTime.Year()] = append(tracksByYear[addedTime.Year()], track.ID)
	}
	return tracksByYear
}

// look at go for each to parse through whole list ->
// map year -> track Array.
// then iterate over the map.
// add song in new playlist for each user.
func createPlayListBasedOnYear(client *spotify.Client, trackIDs []spotify.ID, year int) {
	res, err := client.CreatePlaylistForUser(context.Background(), "12153283982", fmt.Sprintf("RohitKatyal-Liked-%d", year), "", false, false)
	if err != nil {
		print(err.Error())
	}
	print(res.ID)
	_, err = client.AddTracksToPlaylist(context.Background(), res.ID, trackIDs...)
	if err != nil {
		print(err.Error())
	}
	print(res.Name)
}

// # Not helpful as most genre objects were just empty
// func getGenreForTrack(track spotify.SavedTrack, client spotify.Client) {
// 	fmt.Println("genre called with "+ track.Album.ID.String())
// 	res, err := client.GetAlbum(context.Background(), track.Album.ID,)
// 	if err != nil {
// 		print("error in gettng genre {} due to error=" + err.Error())
// 	}
// }

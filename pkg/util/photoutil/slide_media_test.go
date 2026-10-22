package photoutil_test

import (
	"context"
	"slices"
	"testing"

	"github.com/autobutler-org/quark/pkg/util/photoutil"
)

func TestListPhotos_SkipsPicturesOfImportedPresentations(t *testing.T) {
	registry := newPhotoRegistry(t,
		"Talk.qslide",
		"Talk_media/image1.png",
		"Talk_media/image2.jpeg",
		"work/Pitch_(1).qslide",
		"work/Pitch_media/image1.png",
		"work/Deck_(2)_media/image1.png",
		"work/Deck_(2).qslide",
		"Trip_media/beach.jpg",
		"holiday.jpg",
	)

	result, err := photoutil.ListPhotos(photoutil.ListPhotosParams{
		Ctx: context.Background(), Registry: registry, Access: systemAccess(t),
		Sort: photoutil.SortName, Order: photoutil.OrderAsc, Limit: 50,
	})
	if err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, p := range result.Photos {
		got = append(got, p.RelPath)
	}
	want := []string{"Trip_media/beach.jpg", "holiday.jpg"}
	slices.Sort(got)
	if !slices.Equal(got, want) || result.Total != len(want) {
		t.Fatalf("photos = %v (total %d), want %v: a presentation's pictures are not photos", got, result.Total, want)
	}
}

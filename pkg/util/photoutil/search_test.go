package photoutil_test

import (
	"context"
	"testing"

	"github.com/autobutler-org/quark/pkg/util/photoutil"
)

func TestListPhotos_QueryPagesTheMatches(t *testing.T) {
	registry := newPhotoRegistry(t,
		"beach/plain.jpg",
		"Beach_a.jpg",
		"beach_b.jpg",
		"other.jpg",
		"beach_notes.txt",
	)

	page, err := photoutil.ListPhotos(photoutil.ListPhotosParams{
		Ctx: context.Background(), Registry: registry, Access: systemAccess(t),
		Query: "BEACH", Sort: photoutil.SortName, Order: photoutil.OrderAsc,
		Offset: 1, Limit: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || len(page.Photos) != 1 || page.Photos[0].FileName != "beach_b.jpg" {
		t.Fatalf("page = %+v, want total 2 and beach_b.jpg", page)
	}

	all, err := photoutil.ListPhotos(photoutil.ListPhotosParams{
		Ctx: context.Background(), Registry: registry, Access: systemAccess(t),
		Query: "  ", Offset: 0, Limit: 50,
	})
	if err != nil {
		t.Fatal(err)
	}
	if all.Total != 4 {
		t.Fatalf("blank query total = %d, want the 4 images", all.Total)
	}
}

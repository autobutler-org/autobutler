# Photos Journeys

Covers the Photos page (`/photos`), including Cirrus photos, mobile device photos, albums, and favorites.

---

### JN-PH-001: Browse Cirrus photos

**Preconditions:** Image files exist in Cirrus.

**Steps:**
1. Navigate to `/photos`.
2. Ensure the **Cirrus** or **All** category tab is selected.

**Expected result:**
- Thumbnail grid of images stored on the butler is displayed.
- Images load with correct thumbnails.

---

### JN-PH-002: Browse mobile device photos (mobile only)

**Preconditions:** Running on mobile (iOS or Android). Photo library permission is granted.

**Steps:**
1. Navigate to `/photos`.
2. Select the **Mobile** category tab.

**Expected result:**
- Thumbnails from the device photo library appear.
- Permission prompt appears if permission has not yet been granted; granting it shows photos.

---

### JN-PH-003: Open a photo in the image viewer

**Preconditions:** Photos are visible in the grid (JN-PH-001).

**Steps:**
1. Tap a photo thumbnail.

**Expected result:**
- Full-resolution image opens in the image viewer.
- User can pan and zoom.
- Navigating back returns to the photos grid.

---

### JN-PH-004: Paginate through Cirrus photos

**Preconditions:** More than one page of Cirrus photos exists (> page size, default 50).

**Steps:**
1. Navigate to `/photos`.
2. Scroll to the bottom of the grid.

**Expected result:**
- Additional photos load automatically (infinite scroll / pagination).
- No duplicate items appear.

---

### JN-PH-005: Adjust grid column count

**Preconditions:** User is on the Photos page.

**Steps:**
1. Use the column-count slider or pinch-to-zoom gesture to increase/decrease columns.

**Expected result:**
- Grid reflows with the new column count (min 1, max 8).
- Thumbnails resize proportionally.

---

### JN-PH-006: Upload a photo from the device

**Preconditions:** Running on mobile with photo library permission granted.

**Steps:**
1. Navigate to `/photos`.
2. Tap the upload/FAB button.
3. Select one or more photos from the device library.

**Expected result:**
- Upload progress is shown.
- After completion, the uploaded photo(s) appear in the Cirrus category.

---

### JN-PH-007: Mark a photo as a favorite

**Preconditions:** At least one photo is visible in the grid.

**Steps:**
1. Long-press a photo (or open its context menu).
2. Select **Favorite** (or tap the heart icon).

**Expected result:**
- Photo is added to the Favorites category.
- Favorite indicator (heart icon) is visible on the thumbnail.

---

### JN-PH-008: View favorites

**Preconditions:** At least one photo has been favorited (JN-PH-007).

**Steps:**
1. Navigate to `/photos`.
2. Select the **Favorites** category tab.

**Expected result:**
- Only favorited photos are shown.
- Non-favorited photos are not visible.

---

### JN-PH-009: Remove a photo from favorites

**Preconditions:** At least one photo is favorited (JN-PH-007).

**Steps:**
1. Navigate to `/photos` → Favorites.
2. Long-press the favorited photo (or open context menu).
3. Select **Remove from favorites**.

**Expected result:**
- Photo is removed from the Favorites listing.
- It still appears under its original category (Cirrus / Mobile / All).

---

### JN-PH-010: Create an album

**Preconditions:** User is on the Photos page.

**Steps:**
1. Open the album sidebar.
2. Tap **New album**.
3. Enter an album name.
4. Confirm.

**Expected result:**
- New album appears in the sidebar.
- Album is initially empty.

---

### JN-PH-011: Add photos to an album

**Preconditions:** An album exists (JN-PH-010). Photos are visible.

**Steps:**
1. Open the album in the sidebar to enter "adding to album" mode.
2. Select one or more photos from the grid.
3. Confirm the selection.

**Expected result:**
- Selected photos are associated with the album.
- Album shows the correct count of photos.

---

### JN-PH-012: View an album

**Preconditions:** An album with at least one photo exists (JN-PH-011).

**Steps:**
1. Open the album sidebar.
2. Tap an album.

**Expected result:**
- App navigates to the album view (`AlbumPage`).
- Only photos in the album are shown.

---

### JN-PH-013: View photo metadata

**Preconditions:** A Cirrus photo is open in the image viewer.

**Steps:**
1. Open the info panel or tap the info icon.

**Expected result:**
- Metadata is shown: filename, date, dimensions, size, etc.

---

### JN-PH-014: Rotate a Cirrus photo

**Preconditions:** A Cirrus photo is open.

**Steps:**
1. Open the rotate action (toolbar or context menu).
2. Tap rotate left or rotate right.

**Expected result:**
- Photo is rotated and the change is persisted on the butler.
- Thumbnail in the grid reflects the new orientation.

---

### JN-PH-015: Copy a Cirrus photo

**Preconditions:** A Cirrus photo exists.

**Steps:**
1. Open context menu on the photo.
2. Select **Copy**.
3. Choose a destination folder.

**Expected result:**
- A copy of the photo appears in the destination.
- Original is unchanged.

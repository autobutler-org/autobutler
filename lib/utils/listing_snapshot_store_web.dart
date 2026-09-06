import 'package:quark/utils/listing_snapshot.dart';

/// The web build has no cold launch to speed up: a reload fetches fresh, so
/// nothing is kept.
const ListingSnapshotStore listingSnapshotStorePlatform =
    NoopListingSnapshotStore();

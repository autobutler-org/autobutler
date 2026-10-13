/// Where one user left off in one video, as `/api/v0/playback` describes it.
class PlaybackPosition {
  const PlaybackPosition({
    this.position = Duration.zero,
    this.duration = Duration.zero,
    this.watched = false,
    this.updatedAt,
  });

  /// How far into the video playback had reached; zero when nothing is stored
  /// or the video is watched.
  final Duration position;

  /// The video's length as the player last reported it.
  final Duration duration;

  /// Whether the video counts as watched: marked by hand, or played to near
  /// its end, which the Quark decides.
  final bool watched;

  /// When the state was last stored; null when the Quark holds nothing.
  final DateTime? updatedAt;

  factory PlaybackPosition.fromJson(Map<String, dynamic> json) =>
      PlaybackPosition(
        position: Duration(
          milliseconds: (json['positionMs'] as num?)?.toInt() ?? 0,
        ),
        duration: Duration(
          milliseconds: (json['durationMs'] as num?)?.toInt() ?? 0,
        ),
        watched: json['watched'] == true,
        updatedAt: DateTime.tryParse(json['updatedAt'] as String? ?? ''),
      );
}

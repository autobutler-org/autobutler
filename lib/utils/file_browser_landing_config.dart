/// Tuning for where the file browser opens: how long a bare `/files` waits
/// for `/auth/status` to say whether the signed-in account is an admin.
class FileBrowserLandingConfig {
  const FileBrowserLandingConfig._();

  /// The longest a bare `/files` holds its first listing for `/auth/status`
  /// (#3211). An admin lands at the root and a member in their home, and the
  /// admin flag is not saved between launches. Longer than
  /// `kAuthRequestTimeout`, so a request that times out is reported as its
  /// own failure; this only bounds a refresh that never settles.
  static const Duration accountWait = Duration(seconds: 6);
}

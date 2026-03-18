import 'dart:convert';

import 'package:autobutler/services/app_settings.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:http/http.dart' as http;

class SbomDependency {
  const SbomDependency({
    required this.path,
    required this.version,
    this.sum,
    this.replace,
  });

  factory SbomDependency.fromJson(Map<String, dynamic> json) {
    return SbomDependency(
      path: json['path'] as String,
      version: json['version'] as String,
      sum: json['sum'] as String?,
      replace: json['replace'] != null
          ? SbomDependency.fromJson(json['replace'] as Map<String, dynamic>)
          : null,
    );
  }

  final String path;
  final String version;
  final String? sum;
  final SbomDependency? replace;
}

class GoSbom {
  const GoSbom({
    required this.goVersion,
    required this.mainPath,
    required this.mainVersion,
    required this.dependencies,
  });

  factory GoSbom.fromJson(Map<String, dynamic> json) {
    final main = json['main'] as Map<String, dynamic>;
    final deps = (json['dependencies'] as List<dynamic>)
        .cast<Map<String, dynamic>>()
        .map(SbomDependency.fromJson)
        .toList(growable: false);
    return GoSbom(
      goVersion: json['goVersion'] as String,
      mainPath: main['path'] as String,
      mainVersion: main['version'] as String,
      dependencies: deps,
    );
  }

  final String goVersion;
  final String mainPath;
  final String mainVersion;
  final List<SbomDependency> dependencies;
}

class FlutterPackage {
  const FlutterPackage({
    required this.name,
    required this.version,
    required this.source,
    this.url,
  });

  factory FlutterPackage.fromJson(Map<String, dynamic> json) {
    return FlutterPackage(
      name: json['name'] as String,
      version: json['version'] as String,
      source: json['source'] as String,
      url: json['url'] as String?,
    );
  }

  final String name;
  final String version;
  final String source;
  final String? url;
}

class SbomService {
  static Uri get _apiBaseUri {
    final configured = AppSettings.instance.activeHost;
    final base = configured ??
        String.fromEnvironment(
          'API_BASE_URL',
          defaultValue: 'http://localhost:8080',
        );
    final uri = Uri.parse(base);
    final isLoopback =
        uri.host == 'localhost' || uri.host == '127.0.0.1' || uri.host == '::1';
    if (!kIsWeb &&
        defaultTargetPlatform == TargetPlatform.android &&
        isLoopback) {
      return uri.replace(host: '10.0.2.2');
    }
    return uri;
  }

  /// Fetch Go SBOM from the backend.
  static Future<GoSbom> getGoSbom() async {
    final uri = _apiBaseUri.resolve('/api/v1/sbom');
    final response = await http.get(uri);
    if (response.statusCode != 200) {
      throw Exception('Failed to load Go SBOM: ${response.statusCode}');
    }
    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return GoSbom.fromJson(json['data'] as Map<String, dynamic>);
  }

  /// Load Flutter SBOM from the embedded asset.
  static Future<List<FlutterPackage>> getFlutterSbom() async {
    final raw = await rootBundle.loadString('assets/sbom_flutter.json');
    final json = jsonDecode(raw) as Map<String, dynamic>;
    final packages = (json['packages'] as List<dynamic>)
        .cast<Map<String, dynamic>>()
        .map(FlutterPackage.fromJson)
        .toList(growable: false);
    return packages;
  }
}

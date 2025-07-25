// web/lib/src/services/api_service.dart
import 'dart:convert';
import 'package:http/http.dart' as http;
import 'package:github_trending_app/src/models/repository.dart';

class ApiService {
  final String baseUrl;

  ApiService({this.baseUrl = 'http://192.168.1.106:8080'});

  Future<Map<String, dynamic>> fetchRepositories({
    String? sessionId,
    List<String>? languages,
    List<String>? tags,
    int page = 0,
  }) async {
    final uri = Uri.parse('$baseUrl/retrieveList').replace(queryParameters: {
      if (sessionId != null) 'sessionId': sessionId,
      if (languages != null && languages.isNotEmpty) 'languages': languages.join(','),
      if (tags != null && tags.isNotEmpty) 'tags': tags.join(','),
      'page': page.toString(),
    });

    final response = await http.get(uri);

    if (response.statusCode == 200) {
      final Map<String, dynamic> data = json.decode(response.body);
      return data;
    } else {
      throw Exception('Failed to load repositories');
    }
  }

  Future<void> trackOpenRepository(String sessionId, int repositoryId) async {
    final uri = Uri.parse('$baseUrl/trackOpenRepository');
    final response = await http.post(
      uri,
      headers: {'Content-Type': 'application/json'},
      body: json.encode({'sessionId': sessionId, 'repositoryId': repositoryId}),
    );

    if (response.statusCode != 200) {
      throw Exception('Failed to track repository open event');
    }
  }
}
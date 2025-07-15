import 'dart:convert';
import 'package:http/http.dart' as http;
import 'package:github_trending_app/models/repository.dart';

class ApiService {
  final String _baseUrl = 'http://localhost:8080';

  Future<List<Repository>> retrieveList(String sessionId, List<String> languages, List<String> tags, int page) async {
    final response = await http.get(Uri.parse('$_baseUrl/retrieveList?sessionId=$sessionId&languages=${languages.join(',')}&tags=${tags.join(',')}&page=$page'));

    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      final List<dynamic> repos = data['repositories'];
      return repos.map((repoData) => Repository.fromMap(repoData)).toList();
    } else {
      throw Exception('Failed to load repositories');
    }
  }

  Future<void> trackOpenRepository(String sessionId, int repositoryId) async {
    await http.post(
      Uri.parse('$_baseUrl/trackOpenRepository'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'sessionId': sessionId,
        'repositoryId': repositoryId,
      }),
    );
  }

  Future<String> fetchReadme(String owner, String repo, String defaultBranch) async {
    final response = await http.get(Uri.parse('https://raw.githubusercontent.com/$owner/$repo/$defaultBranch/README.md'));
    if (response.statusCode == 200) {
      return response.body;
    } else {
      return 'Failed to load README';
    }
  }
}

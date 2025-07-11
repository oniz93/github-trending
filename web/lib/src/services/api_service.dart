import 'dart:convert';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:http/http.dart' as http;
import 'package:github_trending_app/src/providers/session_provider.dart';

final apiServiceProvider = Provider<ApiService>((ref) {
  return ApiService(ref);
});

class ApiService {
  final Ref _ref;
  final String _baseUrl = 'http://localhost:8080'; // Adjust if your API is hosted elsewhere

  ApiService(this._ref);

  Future<Map<String, dynamic>> retrieveList() async {
    final sessionState = _ref.read(sessionProvider);
    final sessionId = sessionState.sessionId;

    final uri = Uri.parse('$_baseUrl/retrieveList').replace(
      queryParameters: {
        if (sessionId != null) 'sessionId': sessionId,
      },
    );

    final response = await http.get(uri);

    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      await _ref.read(sessionProvider.notifier).updateSession(data['sessionId']);
      return data;
    } else {
      throw Exception('Failed to retrieve list');
    }
  }

  Future<void> trackOpenRepository(int repositoryId) async {
    final sessionId = _ref.read(sessionProvider).sessionId;
    if (sessionId == null) return; // Or handle error

    final uri = Uri.parse('$_baseUrl/trackOpenRepository');
    await http.post(
      uri,
      headers: {'Content-Type': 'application/json'},
      body: json.encode({
        'sessionId': sessionId,
        'repositoryId': repositoryId,
      }),
    );
  }
}

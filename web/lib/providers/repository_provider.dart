// web/lib/src/providers/repository_provider.dart
import 'package:flutter/material.dart';
import '../models/repository.dart';
import '../services/api_service.dart';

class RepositoryProvider with ChangeNotifier {
  final ApiService _apiService;
  List<Repository> _repositories = [];
  bool _isLoading = false;
  String? _errorMessage;
  String? _sessionId;
  int _currentPage = 0;

  List<Repository> get repositories => _repositories;
  bool get isLoading => _isLoading;
  String? get errorMessage => _errorMessage;
  String? get sessionId => _sessionId;

  RepositoryProvider({ApiService? apiService}) : _apiService = apiService ?? ApiService();

  Future<void> fetchRepositories({
    List<String>? languages,
    List<String>? tags,
    bool refresh = false,
  }) async {
    if (_isLoading) return;

    _isLoading = true;
    _errorMessage = null;
    if (refresh) {
      _currentPage = 0;
      _repositories = [];
    }
    notifyListeners();

    try {
      final responseData = await _apiService.fetchRepositories(
        sessionId: _sessionId,
        languages: languages,
        tags: tags,
        page: _currentPage,
      );

      final List<dynamic> repositoriesJson = responseData['repositories'];
      final String? newSessionId = responseData['sessionId'];

      if (newSessionId != null && _sessionId == null) {
        _sessionId = newSessionId;
      }

      _repositories.addAll(repositoriesJson.map((json) => Repository.fromJson(json)).toList());
      _currentPage++;
    } catch (e) {
      _errorMessage = 'Failed to load repositories: ${e.toString()}';
      print(_errorMessage); // Log the error for debugging
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  Future<void> trackOpenRepository(int repositoryId) async {
    if (_sessionId == null) {
      print("Warning: No session ID available to track repository open event.");
      return;
    }
    try {
      await _apiService.trackOpenRepository(_sessionId!, repositoryId);
    } catch (e) {
      print('Error tracking repository open event: ${e.toString()}');
    }
  }

  void setSessionId(String id) {
    _sessionId = id;
    notifyListeners();
  }
}
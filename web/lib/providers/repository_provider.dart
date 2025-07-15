import 'package:flutter/material.dart';
import 'package:github_trending_app/models/repository.dart';
import 'package:github_trending_app/services/api_service.dart';
import 'package:uuid/uuid.dart';

class RepositoryProvider with ChangeNotifier {
  final ApiService _apiService = ApiService();

  List<Repository> _repositories = [];
  List<Repository> get repositories => _repositories;

  bool _isLoading = false;
  bool get isLoading => _isLoading;

  int _page = 0;
  String _sessionId = Uuid().v4();

  Future<void> fetchRepositories({List<String> languages = const [], List<String> tags = const []}) async {
    _isLoading = true;
    notifyListeners();

    final newRepositories = await _apiService.retrieveList(
      _sessionId,
      languages,
      tags,
      _page,
    );

    _repositories.addAll(newRepositories);
    _page++;
    _isLoading = false;
    notifyListeners();
  }

  Future<void> trackOpenRepository(int repositoryId) async {
    await _apiService.trackOpenRepository(_sessionId, repositoryId);
  }

  Future<void> fetchReadmeForRepository(Repository repository) async {
    final readme = await _apiService.fetchReadme(
      repository.owner.login,
      repository.name,
      repository.defaultBranch,
    );
    final index = _repositories.indexWhere((repo) => repo.id == repository.id);
    if (index != -1) {
      _repositories[index] = _repositories[index].copyWith(readmeSnippet: readme);
      notifyListeners();
    }
  }
}

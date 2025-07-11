import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:github_trending_app/src/services/api_service.dart';

final repositoryProvider = StateNotifierProvider<RepositoryNotifier, AsyncValue<List<dynamic>>>((ref) {
  return RepositoryNotifier(ref);
});

class RepositoryNotifier extends StateNotifier<AsyncValue<List<dynamic>>> {
  final Ref _ref;
  bool _isLoading = false;

  RepositoryNotifier(this._ref) : super(const AsyncValue.loading());

  Future<void> fetchRepositories() async {
    if (_isLoading) return;
    _isLoading = true;

    try {
      final data = await _ref.read(apiServiceProvider).retrieveList();
      final newRepositories = data['repositories'] as List<dynamic>;

      final currentState = state;
      if (currentState is AsyncData<List<dynamic>>) {
        state = AsyncValue.data([...currentState.value, ...newRepositories]);
      } else {
        state = AsyncValue.data(newRepositories);
      }
    } catch (e, s) {
      state = AsyncValue.error(e, s);
    }

    _isLoading = false;
  }
}

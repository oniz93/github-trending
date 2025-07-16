import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

final sessionProvider = StateNotifierProvider<SessionNotifier, SessionState>((ref) {
  return SessionNotifier();
});

class SessionState {
  final String? sessionId;
  final DateTime? lastUsed;

  SessionState({this.sessionId, this.lastUsed});
}

class SessionNotifier extends StateNotifier<SessionState> {
  SessionNotifier() : super(SessionState()) {
    _loadSession();
  }

  Future<void> _loadSession() async {
    final prefs = await SharedPreferences.getInstance();
    final sessionId = prefs.getString('sessionId');
    final lastUsedMillis = prefs.getInt('lastUsed');

    if (sessionId != null && lastUsedMillis != null) {
      final lastUsed = DateTime.fromMillisecondsSinceEpoch(lastUsedMillis);
      if (DateTime.now().difference(lastUsed) < const Duration(hours: 1)) {
        state = SessionState(sessionId: sessionId, lastUsed: lastUsed);
      }
    }
  }

  Future<void> updateSession(String newSessionId) async {
    final prefs = await SharedPreferences.getInstance();
    final now = DateTime.now();
    await prefs.setString('sessionId', newSessionId);
    await prefs.setInt('lastUsed', now.millisecondsSinceEpoch);
    state = SessionState(sessionId: newSessionId, lastUsed: now);
  }
}

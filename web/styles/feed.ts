import { StyleSheet } from 'react-native';

export const feedStyles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#000',
  },
  gradient: {
    position: 'absolute',
    left: 0,
    right: 0,
    top: 0,
    bottom: 0,
  },
  topBar: {
    position: 'absolute',
    top: 40,
    right: 16,
    left: 16,
    flexDirection: 'row',
    justifyContent: 'flex-end',
    alignItems: 'center',
    zIndex: 1,
  },
  topBarButton: {
    paddingVertical: 8,
    paddingHorizontal: 16,
    borderRadius: 6,
    backgroundColor: 'rgba(33, 38, 45, 0.8)',
    marginLeft: 8,
    backdropFilter: 'blur(10px)',
  },
  topBarButtonText: {
    color: 'white',
    fontWeight: 'bold',
  },
  loadingContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: '#0d1117',
  },
  loadingText: {
    color: 'white',
    marginTop: 8,
  },
  projectContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
});

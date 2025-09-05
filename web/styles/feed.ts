import { StyleSheet } from 'react-native';

export const feedStyles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#0d1117',
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
    display: 'none',
  },
  topBarButtonText: {
    color: 'white',
    fontWeight: 'bold',
  },
  loadingContainer: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    justifyContent: 'flex-end',
    alignItems: 'center',
    zIndex: 2,
    paddingBottom: '15%',
  },
  apologyText: {
    color: 'white',
    marginTop: 10,
    fontSize: 16,
  },
  projectContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
});

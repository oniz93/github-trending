import React from 'react';
import { View, Text, StyleSheet } from 'react-native';

const RepositoryIndexScreen = () => {
  return (
    <View style={styles.container}>
      <Text style={styles.text}>Please specify a repository ID.</Text>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: '#0d1117',
  },
  text: {
    color: 'white',
    fontSize: 18,
  },
});

export default RepositoryIndexScreen;

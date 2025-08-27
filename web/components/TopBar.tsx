import React from 'react';
import { View, Text, StyleSheet, TouchableOpacity } from 'react-native';

const TopBar = () => {
  return (
    <View style={styles.container}>
      <TouchableOpacity style={styles.button}>
        <Text style={styles.buttonText}>Login</Text>
      </TouchableOpacity>
      <TouchableOpacity style={[styles.button, styles.githubButton]}>
        <Text style={styles.buttonText}>Login with GitHub</Text>
      </TouchableOpacity>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    position: 'absolute',
    top: 40,
    right: 16,
    left: 16,
    flexDirection: 'row',
    justifyContent: 'flex-end',
    alignItems: 'center',
    zIndex: 1,
  },
  button: {
    paddingVertical: 8,
    paddingHorizontal: 16,
    borderRadius: 6,
    backgroundColor: '#21262d',
    marginLeft: 8,
  },
  githubButton: {
    backgroundColor: '#238636',
  },
  buttonText: {
    color: 'white',
    fontWeight: 'bold',
  },
});

export default TopBar;

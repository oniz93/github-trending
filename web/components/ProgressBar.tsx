
import React, { useEffect, useState } from 'react';
import { View, StyleSheet } from 'react-native';

interface ProgressBarProps {
  duration: number;
  onFinish: () => void;
}

const ProgressBar: React.FC<ProgressBarProps> = ({ duration, onFinish }) => {
  const [progress, setProgress] = useState(0);

  useEffect(() => {
    const interval = setInterval(() => {
      setProgress(prev => {
        if (prev >= 100) {
          clearInterval(interval);
          onFinish();
          return 100;
        }
        return prev + 100 / (duration / 100);
      });
    }, 100);

    return () => clearInterval(interval);
  }, [duration, onFinish]);

  return (
    <View style={styles.progressBarContainer}>
      <View style={[styles.progressBar, { width: `${progress}%` }]} />
    </View>
  );
};

const styles = StyleSheet.create({
  progressBarContainer: {
    height: 4,
    width: '80%',
    backgroundColor: 'rgba(255, 255, 255, 0.2)',
    borderRadius: 2,
    overflow: 'hidden',
    alignSelf: 'center',
    position: 'absolute',
    bottom: '20%',
  },
  progressBar: {
    height: '100%',
    backgroundColor: '#58a6ff',
    borderRadius: 2,
  },
});

export default ProgressBar;

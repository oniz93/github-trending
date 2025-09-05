import React from 'react';
import { View, Text, StyleSheet, Platform } from 'react-native';

interface TooltipProps {
  text: string;
}

const Tooltip: React.FC<TooltipProps> = ({ text }) => {
  return (
    <View style={styles.tooltipContainer}>
      <Text style={styles.tooltipText}>{text}</Text>
      <View style={styles.tooltipArrow} />
    </View>
  );
};

const styles = StyleSheet.create({
  tooltipContainer: {
    position: 'absolute',
    top: -40,
    backgroundColor: '#333',
    borderRadius: 6,
    paddingVertical: 6,
    paddingHorizontal: 10,
    ...Platform.select({
        ios: {
            shadowColor: '#000',
            shadowOffset: { width: 0, height: 2 },
            shadowOpacity: 0.8,
            shadowRadius: 2,
        },
        android: {
            elevation: 5,
        },
        web: {
            boxShadow: '0 2px 2px rgba(0, 0, 0, 0.8)',
        }
    }),
  },
  tooltipText: {
    color: 'white',
    fontSize: 14,
  },
  tooltipArrow: {
    position: 'absolute',
    bottom: -5,
    left: '50%',
    marginLeft: -5,
    width: 0,
    height: 0,
    borderLeftWidth: 5,
    borderRightWidth: 5,
    borderTopWidth: 5,
    borderLeftColor: 'transparent',
    borderRightColor: 'transparent',
    borderTopColor: '#333',
  },
});

export default Tooltip;

import React, { useState, useCallback } from 'react';
import { Text, StyleSheet, View } from 'react-native';

interface AutoShrinkingTextProps {
  text: string;
  style: any;
}

const AutoShrinkingText: React.FC<AutoShrinkingTextProps> = ({ text, style }) => {
  const [fontSize, setFontSize] = useState(style.fontSize || 14);

  const onTextLayout = useCallback(e => {
    const { width } = e.nativeEvent.layout;
    if (e.nativeEvent.lines && e.nativeEvent.lines.length > 0) {
      const textWidth = e.nativeEvent.lines.reduce((acc, line) => acc + line.width, 0);
      if (textWidth > width && fontSize > 8) {
        setFontSize(fontSize - 1);
      }
    }
  }, [fontSize]);

  return (
    <Text style={[style, { fontSize }]} onLayout={onTextLayout}>
      {text}
    </Text>
  );
};

export default AutoShrinkingText;

import React from 'react';
import { View, Text, StyleSheet, TouchableOpacity } from 'react-native';
import { FeedProject } from '../types/repository';

interface SpecialMessageCardProps {
  project: FeedProject;
}

const SpecialMessageCard: React.FC<SpecialMessageCardProps> = ({ project }) => {
  const isGetFeaturedMessage = project.name.toLowerCase().includes('featured');

  return (
    <View style={styles.cardContainer}>
      {isGetFeaturedMessage && <View style={styles.glow} />}
      <View style={[styles.innerContainer, isGetFeaturedMessage && styles.featuredInnerContainer]}>
        <View style={styles.contentContainer}>
          {isGetFeaturedMessage ? (
            <>
              <View style={styles.featuredBadge}>
                <Text style={styles.featuredIcon}>✨</Text>
                <Text style={styles.featuredText}>Get Featured</Text>
                <Text style={styles.featuredIcon}>✨</Text>
              </View>
              <Text style={styles.title}>{project.name}</Text>
              <Text style={styles.description}>{project.description}</Text>
              <View style={styles.featuredContent}>
                <Text style={styles.featuredHowTo}>How to get featured:</Text>
                <Text style={styles.featuredStep}>- Star our repository on GitHub</Text>
                <Text style={styles.featuredStep}>- We also appreciate if you create a pull request</Text>
                <Text style={styles.featuredStep}>- Consider sharing GitTok on social media</Text>
              </View>
              <View style={styles.buttonContainer}>
                <TouchableOpacity style={styles.actionButton}>
                    <Text style={styles.actionButtonText}>Star on GitHub</Text>
                </TouchableOpacity>
                <TouchableOpacity style={styles.actionButton}>
                    <Text style={styles.actionButtonText}>Share on Twitter</Text>
                </TouchableOpacity>
              </View>
            </>
          ) : (
            <>
              <Text style={styles.title}>{project.name}</Text>
              <Text style={styles.description}>{project.description}</Text>
              {/* <TouchableOpacity style={styles.actionButton}>
                <Text style={styles.actionButtonText}>Learn More</Text>
              </TouchableOpacity> */}
            </>
          )}
        </View>
      </View>
    </View>
  );
};

const styles = StyleSheet.create({
    cardContainer: {
        flex: 1,
        justifyContent: 'center',
        alignItems: 'center',
      },
      glow: {
        position: 'absolute',
        top: -2, left: -2, right: -2, bottom: -2,
        borderRadius: 14,
        backgroundColor: 'purple',
        opacity: 0.4,
        filter: 'blur(10px)',
      },
      innerContainer: {
        width: '100%',
        maxWidth: 500,
        backgroundColor: 'rgba(13, 17, 23, 0.8)',
        borderRadius: 12,
        padding: 24,
        alignItems: 'center',
      },
      featuredInnerContainer: {
        backgroundColor: 'rgba(13, 17, 23, 0.9)',
        borderColor: 'purple',
        borderWidth: 1,
      },
      contentContainer: {
        alignItems: 'center',
      },
      featuredBadge: {
        flexDirection: 'row',
        alignItems: 'center',
        backgroundColor: 'purple',
        paddingVertical: 6,
        paddingHorizontal: 12,
        borderRadius: 16,
        marginBottom: 16,
      },
      featuredIcon: {
        fontSize: 14,
        marginHorizontal: 4,
      },
      featuredText: {
        color: 'white',
        fontWeight: 'bold',
        fontSize: 14,
      },
      title: {
        fontFamily: 'serif',
        fontSize: 28,
        color: 'white',
        fontWeight: 'bold',
        textAlign: 'center',
        marginBottom: 8,
      },
      description: {
        fontFamily: 'serif',
        fontSize: 18,
        color: '#e1e4e8',
        textAlign: 'center',
        marginBottom: 24,
      },
      featuredContent: {
        backgroundColor: 'rgba(255, 255, 255, 0.05)',
        borderRadius: 8,
        padding: 16,
        marginBottom: 24,
        width: '100%',
      },
      featuredHowTo: {
        color: 'white',
        fontSize: 16,
        fontWeight: 'bold',
        marginBottom: 8,
      },
      featuredStep: {
        color: '#e1e4e8',
        fontSize: 14,
        marginBottom: 4,
      },
      buttonContainer: {
        flexDirection: 'row',
        gap: 16,
      },
      actionButton: {
        backgroundColor: '#58a6ff',
        paddingVertical: 12,
        paddingHorizontal: 24,
        borderRadius: 8,
      },
      actionButtonText: {
        color: 'white',
        fontWeight: 'bold',
        fontSize: 16,
      },
});

export default SpecialMessageCard;

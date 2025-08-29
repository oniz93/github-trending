import { glancify } from 'glancify';
import { NextResponse } from 'next/server';

// Function to resolve shortened URLs
async function resolveUrl(url, maxRedirects = 5) {
  let currentUrl = url;
  let redirectCount = 0;
  
  while (redirectCount < maxRedirects) {
    try {
      console.log(`Resolving URL (attempt ${redirectCount + 1}):`, currentUrl);
      
      const response = await fetch(currentUrl, {
        method: 'HEAD',
        redirect: 'manual',
        headers: {
          'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36'
        }
      });
      
      // Check if it's a redirect
      if (response.status >= 300 && response.status < 400) {
        const location = response.headers.get('location');
        if (location) {
          // Handle relative URLs
          if (location.startsWith('/')) {
            const urlObj = new URL(currentUrl);
            currentUrl = `${urlObj.protocol}//${urlObj.host}${location}`;
          } else if (location.startsWith('http')) {
            currentUrl = location;
          } else {
            // Relative to current path
            const urlObj = new URL(currentUrl);
            currentUrl = new URL(location, urlObj.href).href;
          }
          redirectCount++;
          console.log(`Redirected to:`, currentUrl);
        } else {
          break;
        }
      } else {
        // No more redirects
        break;
      }
    } catch (error) {
      console.error(`Error resolving URL at step ${redirectCount}:`, error.message);
      break;
    }
  }
  
  console.log(`Final resolved URL:`, currentUrl);
  return currentUrl;
}

// Function to check if URL is a known shortener
function isShortenerUrl(url) {
  try {
    const hostname = new URL(url).hostname.toLowerCase();
    const shorteners = [
      't.co',
      'bit.ly',
      'tinyurl.com',
      'goo.gl',
      'ow.ly',
      'buff.ly',
      'short.link',
      'is.gd',
      'v.gd',
      'cutt.ly',
      'rebrand.ly'
    ];
    
    return shorteners.some(shortener => hostname.includes(shortener));
  } catch {
    return false;
  }
}

export async function GET(request) {
  const { searchParams } = new URL(request.url);
  const originalUrl = searchParams.get('url');

  if (!originalUrl) {
    return NextResponse.json({ error: 'URL parameter is required' }, { status: 400 });
  }

  try {
    // Validate original URL
    new URL(originalUrl);
    
    let finalUrl = originalUrl;
    
    // If it's a shortener URL, resolve it first
    if (isShortenerUrl(originalUrl)) {
      console.log('Detected shortener URL, resolving:', originalUrl);
      try {
        finalUrl = await resolveUrl(originalUrl);
        console.log('Resolved to:', finalUrl);
      } catch (resolveError) {
        console.error('Failed to resolve shortened URL:', resolveError.message);
        return NextResponse.json({ 
          error: 'Failed to resolve shortened URL',
          originalUrl: originalUrl,
          details: resolveError.message
        }, { status: 400 });
      }
    }
    
    // Set a timeout for the glancify request
    const timeoutPromise = new Promise((_, reject) => {
      setTimeout(() => reject(new Error('Preview request timeout')), 15000);
    });
    
    const previewPromise = glancify(finalUrl);
    
    const preview = await Promise.race([previewPromise, timeoutPromise]);
    
    // Validate preview data
    if (!preview || typeof preview !== 'object') {
      return NextResponse.json({ 
        error: 'Invalid preview data received',
        originalUrl: originalUrl,
        finalUrl: finalUrl
      }, { status: 500 });
    }
    
    // Clean up the preview data
    const cleanPreview = {
      title: preview.title?.trim() || null,
      description: preview.description?.trim() || null,
      image: preview.image || null,
      originalUrl: originalUrl,
      finalUrl: finalUrl,
      hostname: new URL(finalUrl).hostname,
      wasRedirected: originalUrl !== finalUrl
    };
    
    return NextResponse.json(cleanPreview);
    
  } catch (error) {
    console.error('Preview fetch error for', originalUrl, ':', error.message);
    
    // Return different error messages based on error type
    if (error.message.includes('timeout')) {
      return NextResponse.json(
        { error: 'Preview request timed out', originalUrl: originalUrl }, 
        { status: 408 }
      );
    }
    
    if (error.message.includes('CORS')) {
      return NextResponse.json(
        { error: 'CORS policy blocks preview', originalUrl: originalUrl }, 
        { status: 403 }
      );
    }
    
    return NextResponse.json(
      { error: 'Failed to fetch preview', details: error.message, originalUrl: originalUrl }, 
      { status: 500 }
    );
  }
}
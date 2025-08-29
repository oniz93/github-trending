'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import Image from 'next/image';

export default function Header({ currentPage = 'home' }) {
  const [isScrolled, setIsScrolled] = useState(false);

  useEffect(() => {
    const handleScroll = () => {
      const scrollTop = window.scrollY;
      setIsScrolled(scrollTop > 50);
    };

    window.addEventListener('scroll', handleScroll);
    return () => window.removeEventListener('scroll', handleScroll);
  }, []);

  return (
    <header className={`header ${isScrolled ? 'scrolled' : ''}`}>
      <nav className="nav">
        <div className="nav-brand">
          <Link href="/" className="brand-link">
            <Image 
              src="/images/open-source-projects-dark-mini.png"
              alt="Open-source Projects"
              width={180}
              height={32}
              className="brand-logo"
              priority
              unoptimized
            />
          </Link>
        </div>
        <div className="nav-links">
          <Link href="/" className={`nav-link ${currentPage === 'home' ? 'active' : ''}`}>
            <i className="fas fa-home"></i>
            <span>Home</span>
          </Link>
          <Link href="/bookmarks" className={`nav-link ${currentPage === 'bookmarks' ? 'active' : ''}`}>
            <i className="fas fa-bookmark"></i>
            <span>Bookmarks</span>
          </Link>
          <Link href="/sponsor-us" className={`nav-link ${currentPage === 'sponsor-us' ? 'active' : ''}`}>
            <i className="fas fa-gem"></i>
            <span>Sponsors</span>
          </Link>
          {/* <Link href="#" className="nav-link">
            <i className="fas fa-gem"></i>
            <span>Hidden Gems</span>
          </Link> */}
          <Link href="/newsletter" className={`nav-link ${currentPage === 'newsletter' ? 'active' : ''}`}>
            <i className="fas fa-envelope"></i>
            <span>Newsletter</span>
          </Link>
          <Link href="/rss-info" className={`nav-link ${currentPage === 'rss-info' ? 'active' : ''}`}>
            <i className="fas fa-rss"></i>
            <span>RSS</span>
          </Link>
        </div>
      </nav>
    </header>
  );
}

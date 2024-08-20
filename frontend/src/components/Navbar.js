import React from 'react';
import styles from '../styles/Navbar.module.css';

const Navbar = ({ siteTitle }) => {
    return (
        <nav className={styles.nav}>
            <h1 className={styles.title}>{siteTitle}</h1>
        </nav>
    );
};

export default Navbar;
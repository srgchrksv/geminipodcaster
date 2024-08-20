import React from 'react';
import styles from '../styles/Transcriptions.module.css';

const Transcriptions = ({ transcriptions }) => {
    if (transcriptions.length === 0) {
        return 
    }

    return (
       <>
        <h2 className={styles.title}>Transcriptions</h2>
        <div className={styles.transcriptions}>
        {transcriptions.map((text, index) => (
            (text.includes("Host:")? <li key={index} className={styles.host}>{text}</li> : text.includes("USERS INTERACTION:") ? <li key={index} className={styles.usersInteraction}>{text}</li>: <li key={index} className={styles.guest}>{text}</li>) 
        ))}
        </div>
       </>
    );
};

export default Transcriptions;


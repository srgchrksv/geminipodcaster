import { useState } from 'react';
import Transcriptions from './Transcriptions';
import styles from '../styles/Form.module.css';

export default function Form() {
    const [transcriptions, setTranscriptions] = useState([]);
    const [audioQueue, setAudioQueue] = useState([]);
    const [sessionId, setSessionId] = useState(null);


    const handleSubmitContext = async (event) => {
        event.preventDefault();
        setTranscriptions([]);

        try {
            const response = await fetch('http://localhost:8000/');
            if (!response.ok) {
                console.error('Failed to make HTTP GET request');
            } else {
                const data = await response.json();
                console.log(data);
            }
        } catch (error) {
            console.error('Error:', error);
        }

        try {
            const formData = new FormData(event.target);
            const file = formData.get('text_file'); // Assuming the file input name is 'file'
            console.log(formData);

            if (file && file.type === "text/plain") {
                const reader = new FileReader();
                reader.onload = function(event) {
                    const fileBinary = event.target.result;

                    const socket = new WebSocket("ws://localhost:8000/podcast");
                    socket.binaryType = "arraybuffer"; // Set binary type to arraybuffer
                    socket.onopen = function() {
                        console.log('WebSocket connection established');
                        socket.send(fileBinary); // Send file data as binary
                    };
                    socket.onmessage = (event) => {
                        if (typeof event.data === 'string') {
                            console.log(event.data);
                            setTranscriptions((prevTranscriptions) => [...prevTranscriptions, event.data]);
                        } else if (event.data instanceof Blob) {
                            console.log(event.data);
                        }
                    };
                };
                reader.readAsArrayBuffer(file); // Read file as ArrayBuffer
            } else {
                console.error('No valid text file selected');
            }
        } catch (error) {
            console.error('Error:', error);
        }
    }
    const handleSubmit = async (event) => {
        event.preventDefault();
        try {
            const response = await fetch('http://localhost:8000/', {
                method: 'GET',
            });

            if (!response.ok) {
                console.error('Failed to send interaction');
            } else {
                const data = await response.json()
                console.log(data)
            }
        } catch (error) {
            console.error('Error:', error);
        }
    };
    const handleSubmitSession = async (event) => {
        event.preventDefault();
        try {
            const response = await fetch('http://localhost:8000/session', {
                method: 'GET',
                credentials: 'include',
            });

            if (!response.ok) {
                console.error('Failed to send interaction');
            } else {
                const data = await response.json()
                console.log(data)
            }
        } catch (error) {
            console.error('Error:', error);
        }
    };

    const handleSubmitInteraction = async (event) => {
        event.preventDefault();
        const formData = new FormData(event.target);
        try {
            const response = await fetch('http://localhost:8000/interact', {
                method: 'POST',
                body: formData,
                credentials: 'include',
            });

            if (!response.ok) {
                console.error('Failed to send interaction');
            } else {
                const data = await response.json()
                console.log(data)
            }
        } catch (error) {
            console.error('Error:', error);
        }
    };


    return (
        <>
        
            <div className={styles.container}>
                <form onSubmit={handleSubmitContext} className={styles.form}>
                    <div className={styles.inputGroup}>
                        <label className={styles.label} htmlFor="context">Upload context for the podcast:</label>
                        <input type="file" id="text_file" name="text_file" />
                    </div>
                    <button className={styles.button} type="submit">Submit</button>
                </form>
                <form onSubmit={handleSubmitInteraction} className={styles.form}>
                    <div className={styles.inputGroup}>
                        <label className={styles.label}>Interact with podcast:</label>
                        <textarea type="text" id="user_interaction" name="user_interaction" />
                    </div>
                    <button className={styles.button} type="submit">Send message</button>
                </form>
                <button className={styles.button} onClick={handleSubmit}>GET</button>
                <button className={styles.button} onClick={handleSubmitSession}>SESSION</button>
                <Transcriptions transcriptions={transcriptions} />
                {audioQueue.length > 0 && (<div>
                    {/* <h2>Audio Queue</h2> */}
                    <ul>
                        {audioQueue.map((audioUrl, index) => (
                            <li className={styles.audioList} key={index}>
                                <audio className={styles.audio} controls autoPlay src={audioUrl}></audio>
                            </li>
                        ))}
                    </ul>
                </div>)}
            </div>
        </>
    );
}
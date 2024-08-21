import { useState } from 'react';
import Transcriptions from './Transcriptions';
import styles from '../styles/Form.module.css';
import Podcast from './Podcast';


export default function Form() {
    const [transcriptions, setTranscriptions] = useState([]);
    const [text, setText] = useState('');
    let textTranscriptionQueue = [];
    let audioQueue = [];
    let isProcessingQueues = false;
    const handleSubmitContext = async (event) => {
        event.preventDefault();
        setTranscriptions([]);
        try {
            const response = await fetch('http://localhost:8000/', {
                method: 'GET',
                credentials: 'include',
            });
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
            // console.log(formData);

            if (file && file.type === "text/plain") {
                const reader = new FileReader();
                reader.onload = function (event) {
                    const fileBinary = event.target.result;

                    const socket = new WebSocket("ws://localhost:8000/podcast");
                    socket.binaryType = "arraybuffer"; // Set binary type to arraybuffer
                    socket.onopen = function () {
                        console.log('WebSocket connection established');
                        socket.send(fileBinary); // Send file data as binary
                    };


                    socket.onmessage = (event) => {
                        if (typeof event.data === 'string') {
                            // Add text transcription to the queue
                            const text = event.data
                            if (text.includes("USERS INTERACTION:")){ 
                                setTranscriptions((prevTranscriptions) => [...prevTranscriptions, text]);
                                return
                            }
                            textTranscriptionQueue.push(event.data);
                            processQueues();
                        } else if (typeof event.data === 'object') {
                            // Add audio to the queue
                            const blob = new Blob([event.data]);
                            audioQueue.push(URL.createObjectURL(blob));
                            processQueues();
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
        function processQueues() {
            if (isProcessingQueues) {
                return;
            }

            isProcessingQueues = true;

            // Process text transcription first
            if (textTranscriptionQueue.length > 0) {
                const nextTranscription = textTranscriptionQueue.shift();
                setTranscriptions((prevTranscriptions) => [...prevTranscriptions, nextTranscription]);
            }

            // Then process audio
            if (audioQueue.length > 0) {
                const nextAudioUrl = audioQueue.shift();
                const audio = new Audio(nextAudioUrl);
                audio.onended = () => {
                    console.log('Audio ended');
                    isProcessingQueues = false;
                    processQueues();
                }
                console.log('Audio play');
                audio.play();
            } else {
                isProcessingQueues = false;
            }
        }

    }


    const handleSubmitInteraction = async (event) => {
        event.preventDefault();
        textTranscriptionQueue = []
        audioQueue = []
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
                <h2 className={styles.title}>Transcriptions</h2>
                <Transcriptions transcriptions={transcriptions} />
            </div>
        </>
    );
}
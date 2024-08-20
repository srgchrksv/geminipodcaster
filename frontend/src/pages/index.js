import Form from '../components/Form';
import Navbar from '@/components/Navbar';
import styles from '../styles/Index.module.css';

export default function Home() {
    const siteTitle = "Geminipodcaster";
  return (
    <main className={styles.main}>
      <Navbar siteTitle={siteTitle} />
      <div className={styles.container}>
        <Form />
      </div>
    </main>
  );
}


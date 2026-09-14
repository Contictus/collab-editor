import Link from 'next/link';
import { redirect } from 'next/navigation';
import { getSession } from '../../lib/session';
import { LoginForm } from './login-form';

export default async function LoginPage() {
  if (await getSession()) redirect('/documents');
  return (
    <main>
      <h1>Sign in</h1>
      <LoginForm />
      <p>
        No account? <Link href="/register">Register</Link>
      </p>
    </main>
  );
}

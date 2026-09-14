import Link from 'next/link';
import { redirect } from 'next/navigation';
import { getSession } from '../../lib/session';
import { RegisterForm } from './register-form';

export default async function RegisterPage() {
  if (await getSession()) redirect('/documents');
  return (
    <main>
      <h1>Create account</h1>
      <RegisterForm />
      <p>
        Already have an account? <Link href="/login">Sign in</Link>
      </p>
    </main>
  );
}

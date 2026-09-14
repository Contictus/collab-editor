import { prisma } from 'db';
import { EmailTakenError, loginUser, registerUser } from '../lib/auth-service';

/**
 * Faz 1 integration smoke against the real Postgres. Exercises register/login
 * (argon2 hash + Prisma) end-to-end minus the cookie/HTTP layer. Run:
 *   pnpm --filter web exec tsx scripts/faz1-smoke.ts
 */
const email = `smoke_${Date.now()}@example.com`;
const password = 'super-secret-123';
let failures = 0;
function check(name: string, cond: boolean) {
  console.log(`${cond ? 'PASS' : 'FAIL'}  ${name}`);
  if (!cond) failures++;
}

async function main() {
  const user = await registerUser({ email, password });
  check('register returns user with id', Boolean(user.id) && user.email === email);

  let dupRejected = false;
  try {
    await registerUser({ email, password });
  } catch (e) {
    dupRejected = e instanceof EmailTakenError;
  }
  check('duplicate email rejected', dupRejected);

  check('login with correct password succeeds', (await loginUser({ email, password }))?.id === user.id);
  check('login with wrong password returns null', (await loginUser({ email, password: 'nope' })) === null);
  check(
    'login with unknown email returns null',
    (await loginUser({ email: 'ghost@example.com', password })) === null,
  );

  // cleanup
  await prisma.user.delete({ where: { id: user.id } });
  await prisma.$disconnect();

  console.log(failures === 0 ? '\nALL PASS' : `\n${failures} FAILED`);
  process.exit(failures === 0 ? 0 : 1);
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});

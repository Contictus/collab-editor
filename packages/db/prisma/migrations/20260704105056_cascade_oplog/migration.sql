-- DropForeignKey
ALTER TABLE "DocumentSnapshot" DROP CONSTRAINT "DocumentSnapshot_documentId_fkey";

-- DropForeignKey
ALTER TABLE "DocumentUpdate" DROP CONSTRAINT "DocumentUpdate_documentId_fkey";

-- AddForeignKey
ALTER TABLE "DocumentUpdate" ADD CONSTRAINT "DocumentUpdate_documentId_fkey" FOREIGN KEY ("documentId") REFERENCES "Document"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "DocumentSnapshot" ADD CONSTRAINT "DocumentSnapshot_documentId_fkey" FOREIGN KEY ("documentId") REFERENCES "Document"("id") ON DELETE CASCADE ON UPDATE CASCADE;

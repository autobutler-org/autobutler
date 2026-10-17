package docreadutil

// addText adds text to the paragraph as a run formatted like run, whose
// own Text is ignored.
func (r *blockPlacer) addText(p *paragraph, run Run, s string) error {
	if s == "" {
		return nil
	}
	cost := len(s)
	if !p.open || p.cur != run {
		cost += runOverhead
	}
	if err := r.e.charge(cost); err != nil {
		return err
	}
	if p.open && p.cur != run {
		p.commit()
	}
	p.cur, p.open = run, true
	p.text.WriteString(s)
	return nil
}

// put places a finished block: into the cell being read, or out to the
// writer at the top level.
func (r *blockPlacer) put(b Block) error {
	r.e.tally(b)
	if r.cell != nil {
		*r.cell = append(*r.cell, b)
		return nil
	}
	return r.e.write(b)
}

// omit places an omitted block for what, splitting the paragraph p it
// interrupts, if any.
func (r *blockPlacer) omit(p *paragraph, what string) error {
	r.e.warnings.count(what)
	return r.split(p, Block{Type: BlockOmitted, What: what})
}

// split places b, first placing what paragraph p (nil at block level) has
// collected so far, so b lands where the document has it.
func (r *blockPlacer) split(p *paragraph, b Block) error {
	if p != nil {
		if err := r.flush(p); err != nil {
			return err
		}
		p.placed = true
	}
	if err := r.e.admit(); err != nil {
		return err
	}
	if err := r.e.charge(blockOverhead + len(b.Part) + len(b.Alt)); err != nil {
		return err
	}
	return r.put(b)
}

// commit moves the run being collected into the block.
func (p *paragraph) commit() {
	if !p.open {
		return
	}
	run := p.cur
	run.Text = p.text.String()
	p.block.Runs = append(p.block.Runs, run)
	p.text.Reset()
	p.open = false
}

// flush places what a paragraph has collected so far, before an image or a
// page break splits it. What follows the split continues as a plain
// paragraph, so a list item's marker or a heading is not repeated.
func (r *blockPlacer) flush(p *paragraph) error {
	p.commit()
	if len(p.block.Runs) == 0 {
		return nil
	}
	if !p.admitted {
		if err := r.e.admit(); err != nil {
			return err
		}
	}
	b := p.block
	p.block = Block{Type: BlockParagraph, Align: b.Align}
	p.admitted, p.placed = false, true
	return r.put(b)
}

// closeParagraph places the rest of a paragraph. An empty paragraph is
// placed, since it is vertical space in the document, unless a split has
// already placed something of it.
func (r *blockPlacer) closeParagraph(p *paragraph) error {
	p.commit()
	if len(p.block.Runs) == 0 && p.placed {
		return nil
	}
	if !p.admitted {
		if err := r.e.admit(); err != nil {
			return err
		}
	}
	return r.put(p.block)
}

// pageBreak places a page break, splitting the paragraph. A page break
// inside a table has nowhere to go and is dropped.
func (r *blockPlacer) pageBreak(p *paragraph) error {
	if r.cell != nil {
		return nil
	}
	return r.split(p, Block{Type: BlockPageBreak})
}

// linkedImage places an image the document links to from outside the
// package. Its target is never fetched, nor passed on.
func (r *blockPlacer) linkedImage(p *paragraph, info drawingInfo) error {
	r.e.warnings.count("linked image")
	return r.split(p, Block{Type: BlockImage, External: true, Alt: info.alt, Width: info.width, Height: info.height})
}

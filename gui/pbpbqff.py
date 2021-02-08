import gi
gi.require_version("Gtk", "3.0")
from gi.repository import Gtk

default_molpro = """memory,1,g

gthresh,energy=1.d-12,zero=1.d-22,oneint=1.d-22,twoint=1.d-22;
gthresh,optgrad=1.d-8,optstep=1.d-8;
nocompress;

geometry={
basis={
default,cc-pVTZ-f12
}
set,charge=0
set,spin=0
hf,accuracy=16,energy=1.0d-10
{CCSD(T)-F12,thrden=1.0d-8,thrvar=1.0d-10}
"""

default_pbqff = """program=gocart
queueType=maple
geomType=xyz
geometry={

}
delta=0.005
flags=noopt
deriv=4
sleepint=60
joblimit=1024
chunksize=64
numjobs=8
intder=/ddn/home6/r2533/programs/intder/Intder2005.x
anpass=/ddn/home6/r2533/programs/anpass/anpass_cerebro.x
spectro=/ddn/home6/r2533/programs/spec3jm.ifort-O0.static.x
"""

# what I really want is a bunch of checkboxes/dropdowns, then generate
# the input files. there should also be documentation for the options
# on hover.

class MyWindow(Gtk.Window):
    def __init__(self):
        Gtk.Window.__init__(self, title="pbpbqff")

        self.molpro_buf = Gtk.TextBuffer()
        self.molpro_buf.set_text(default_molpro)
        self.molpro = Gtk.TextView(buffer=self.molpro_buf)
        self.molpro.set_border_window_size(Gtk.TextWindowType.RIGHT, 200)
        self.molpro.set_border_window_size(Gtk.TextWindowType.BOTTOM, 200)

        self.pbqff_buf = Gtk.TextBuffer()
        self.pbqff_buf.set_text(default_pbqff)
        self.pbqff = Gtk.TextView(buffer=self.pbqff_buf)
        self.pbqff.set_border_window_size(Gtk.TextWindowType.RIGHT, 200)
        self.pbqff.set_border_window_size(Gtk.TextWindowType.BOTTOM, 200)

        self.button = Gtk.Button(label="print me")
        self.button.connect("clicked", self.on_button_clicked)

        self.button2 = Gtk.Button(label="print me")
        self.button2.connect("clicked", self.on_button2_clicked)
        
        self.molpro_grid = Gtk.Grid()
        self.molpro_grid.attach(self.molpro, 0, 0, 1, 1)
        self.molpro_grid.attach(self.button, 0, 1, 1, 1)

        self.pbqff_grid = Gtk.Grid()
        self.pbqff_grid.attach(self.pbqff, 0, 0, 1, 1)
        self.pbqff_grid.attach(self.button2, 0, 1, 1, 1)
        
        self.base = Gtk.Notebook()
        self.base.append_page(self.molpro_grid, Gtk.Label(label="Molpro"))
        self.base.append_page(self.pbqff_grid, Gtk.Label(label="pbqff"))
        self.add(self.base)


    def on_button_clicked(self, widget):
        start = self.molpro_buf.get_start_iter()
        end = self.molpro_buf.get_end_iter()
        print(self.molpro_buf.get_text(start, end, True))

    def on_button2_clicked(self, widget):
        start = self.pbqff_buf.get_start_iter()
        end = self.pbqff_buf.get_end_iter()
        print(self.pbqff_buf.get_text(start, end, True))
    
win = MyWindow()
win.connect("destroy", Gtk.main_quit)
win.show_all()
Gtk.main()
